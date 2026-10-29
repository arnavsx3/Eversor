package core

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var SystemActions = map[string]bool{"apt-cache": true, "apt-autoremove": true, "snap-disabled": true, "journals": true}

func RunningKernel() string {
	var uts syscall.Utsname
	if syscall.Uname(&uts) != nil {
		return ""
	}
	var result []byte
	for _, b := range uts.Release {
		if b == 0 {
			break
		}
		result = append(result, byte(b))
	}
	return string(result)
}

func KernelGuard(plan, packages, running string) error {
	if running == "" {
		return errors.New("running kernel could not be verified")
	}
	var kernels []string
	re := regexp.MustCompile(`^ii\s+linux-image-(?:unsigned-)?([0-9][^\s:]*)`)
	for _, line := range strings.Split(packages, "\n") {
		if match := re.FindStringSubmatch(line); len(match) == 2 {
			kernels = append(kernels, match[1])
		}
	}
	if len(kernels) == 0 {
		return errors.New("installed kernels could not be verified")
	}
	sort.Slice(kernels, func(i, j int) bool { return versionLess(kernels[i], kernels[j]) })
	protected := []string{running, kernels[len(kernels)-1]}
	for _, line := range strings.Split(plan, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || (fields[0] != "Remv" && fields[0] != "Purg") {
			continue
		}
		name := strings.Split(fields[1], ":")[0]
		for _, kernel := range protected {
			base := kernel
			if at := strings.LastIndex(base, "-"); at >= 0 {
				base = base[:at]
			}
			if strings.HasSuffix(name, "-"+kernel) || strings.HasSuffix(name, "-"+base) {
				return fmt.Errorf("APT proposed removing protected kernel package %s", name)
			}
		}
	}
	return nil
}

func versionLess(a, b string) bool {
	parts := regexp.MustCompile(`[0-9]+|[^0-9]+`)
	x, y := parts.FindAllString(a, -1), parts.FindAllString(b, -1)
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] == y[i] {
			continue
		}
		n, nerr := strconv.ParseUint(x[i], 10, 64)
		m, merr := strconv.ParseUint(y[i], 10, 64)
		if nerr == nil && merr == nil {
			return n < m
		}
		return x[i] < y[i]
	}
	return len(x) < len(y)
}

func RunSystem(ctx context.Context, ids []string, runner Runner, kernel string) []Result {
	var results []Result
	for _, id := range ids {
		result := Result{ID: id, Title: id, Status: "cleaned"}
		var err error
		if !SystemActions[id] {
			result.Status = "failed"
			result.Detail = "unsupported system action"
			results = append(results, result)
			continue
		}
		switch id {
		case "apt-cache":
			result.Detail, err = runner.Run(ctx, "/usr/bin/apt-get", "clean")
		case "apt-autoremove":
			var plan, packages string
			plan, err = runner.Run(ctx, "/usr/bin/apt-get", "-s", "autoremove", "--purge")
			if err == nil {
				packages, err = runner.Run(ctx, "/usr/bin/dpkg-query", "-W", "-f=${db:Status-Abbrev} ${binary:Package}\n", "linux-image-*")
			}
			if err == nil {
				err = KernelGuard(plan, packages, kernel)
			}
			if err == nil {
				result.Detail, err = runner.Run(ctx, "/usr/bin/apt-get", "-o", "DPkg::Lock::Timeout=60", "-y", "autoremove", "--purge")
			}
		case "snap-disabled":
			var listing string
			listing, err = runner.Run(ctx, "/usr/bin/snap", "list", "--all")
			if err == nil {
				removed := 0
				for _, revision := range ParseSnaps(listing) {
					if !revision.Disabled {
						continue
					}
					var current string
					current, err = runner.Run(ctx, "/usr/bin/snap", "list", "--all", revision.Name)
					if err != nil {
						break
					}
					stillDisabled := false
					for _, now := range ParseSnaps(current) {
						if now.Name == revision.Name && now.Revision == revision.Revision && now.Disabled {
							stillDisabled = true
						}
					}
					if !stillDisabled {
						continue
					}
					var output string
					output, err = runner.Run(ctx, "/usr/bin/snap", "remove", revision.Name, "--revision="+revision.Revision)
					result.Detail += output
					if err != nil {
						break
					}
					removed++
				}
				if removed == 0 && err == nil {
					result.Status = "skipped"
					result.Detail = "no revisions remain eligible for removal"
				}
			}
		case "journals":
			result.Detail, err = runner.Run(ctx, "/usr/bin/journalctl", "--vacuum-time=14d", "--vacuum-size=500M")
		}
		if err != nil {
			result.Status = "failed"
			result.Detail = strings.TrimSpace(result.Detail + "\n" + err.Error())
		}
		results = append(results, result)
	}
	return results
}
