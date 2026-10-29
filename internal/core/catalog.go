package core

// Paths are relative to known XDG/home roots. Adding an application requires a
// supported cache path, running-process guard and preservation tests.
type Rule struct {
	ID, Name                            string
	Commands, Packages, Flatpaks, Watch []string
	Cache, Config, Home                 []string
}

var Catalog = []Rule{
	// Upstream documents these as download/metadata caches, separate from
	// Cargo's installed executables, kubeconfig, and Helm registry credentials.
	{ID: "cargo-archives", Name: "Cargo crate downloads", Commands: []string{"cargo"}, Watch: []string{"cargo", "rustc", "rust-analyzer", "rustup"}, Home: []string{".cargo/registry/cache"}},
	{ID: "kubectl", Name: "Kubernetes client", Commands: []string{"kubectl"}, Watch: []string{"kubectl", "helm", "k9s", "kustomize", "skaffold", "argocd"}, Home: []string{".kube/cache"}},
	{ID: "helm", Name: "Helm repository indexes", Commands: []string{"helm"}, Watch: []string{"helm"}, Cache: []string{"helm/repository"}},
	{ID: "chrome", Name: "Google Chrome", Commands: []string{"google-chrome", "google-chrome-stable"}, Packages: []string{"google-chrome-stable"}, Watch: []string{"chrome", "google-chrome", "google-chrome-stable"}, Cache: []string{"google-chrome"}, Config: []string{"google-chrome/*/Cache", "google-chrome/*/Code Cache", "google-chrome/*/GPUCache"}},
	{ID: "chromium", Name: "Chromium", Commands: []string{"chromium", "chromium-browser"}, Packages: []string{"chromium", "chromium-browser"}, Flatpaks: []string{"org.chromium.Chromium"}, Watch: []string{"chromium", "chrome"}, Cache: []string{"chromium"}, Home: []string{"snap/chromium/common/.cache/chromium", ".var/app/org.chromium.Chromium/cache/chromium"}},
	{ID: "firefox", Name: "Firefox", Commands: []string{"firefox"}, Packages: []string{"firefox"}, Flatpaks: []string{"org.mozilla.firefox"}, Watch: []string{"firefox", "firefox-bin"}, Cache: []string{"mozilla/firefox"}, Home: []string{"snap/firefox/common/.cache/mozilla/firefox", ".var/app/org.mozilla.firefox/cache/mozilla/firefox"}},
	{ID: "brave", Name: "Brave", Commands: []string{"brave-browser"}, Packages: []string{"brave-browser"}, Flatpaks: []string{"com.brave.Browser"}, Watch: []string{"brave", "brave-browser"}, Cache: []string{"BraveSoftware/Brave-Browser"}, Home: []string{".var/app/com.brave.Browser/cache/BraveSoftware/Brave-Browser"}},
	{ID: "code", Name: "Visual Studio Code", Commands: []string{"code"}, Packages: []string{"code"}, Watch: []string{"code"}, Config: []string{"Code/Cache", "Code/Code Cache", "Code/GPUCache", "Code/CachedData", "Code/CachedExtensionVSIXs"}, Cache: []string{"vscode-cpptools"}},
	{ID: "codium", Name: "VSCodium", Commands: []string{"codium"}, Packages: []string{"codium"}, Watch: []string{"codium"}, Config: []string{"VSCodium/Cache", "VSCodium/Code Cache", "VSCodium/GPUCache", "VSCodium/CachedData"}},
	{ID: "codex", Name: "ChatGPT / Codex", Commands: []string{"codex", "chatgpt"}, Packages: []string{"chatgpt", "codex"}, Watch: []string{"codex", "chatgpt"}, Cache: []string{"Codex"}, Config: []string{"Codex/Cache", "Codex/Code Cache", "Codex/GPUCache"}},
	{ID: "claude", Name: "Claude", Commands: []string{"claude"}, Packages: []string{"claude-desktop"}, Watch: []string{"claude", "claude-desktop"}, Config: []string{"Claude/Cache", "Claude/Code Cache", "Claude/GPUCache"}},
	{ID: "antigravity", Name: "Antigravity", Commands: []string{"antigravity"}, Packages: []string{"antigravity"}, Watch: []string{"antigravity"}, Config: []string{"Antigravity/Cache", "Antigravity/Code Cache", "Antigravity/GPUCache"}},
	{ID: "discord", Name: "Discord", Commands: []string{"discord"}, Packages: []string{"discord"}, Watch: []string{"discord"}, Config: []string{"discord/Cache", "discord/Code Cache", "discord/GPUCache"}},
	{ID: "slack", Name: "Slack", Commands: []string{"slack"}, Packages: []string{"slack-desktop"}, Watch: []string{"slack"}, Config: []string{"Slack/Cache", "Slack/Code Cache", "Slack/GPUCache"}},
	{ID: "pip", Name: "Python pip", Commands: []string{"pip", "pip3"}, Watch: []string{"pip", "pip3", "uv", "uvx"}, Cache: []string{"pip"}},
	{ID: "npm", Name: "npm", Commands: []string{"npm"}, Watch: []string{"npm", "pnpm", "yarn"}, Home: []string{".npm/_cacache"}},
	{ID: "yarn", Name: "Yarn", Commands: []string{"yarn"}, Watch: []string{"yarn", "npm", "pnpm"}, Cache: []string{"yarn"}},
	{ID: "pnpm", Name: "pnpm", Commands: []string{"pnpm"}, Watch: []string{"pnpm", "npm", "yarn"}, Cache: []string{"pnpm"}},
	{ID: "trivy", Name: "Trivy", Commands: []string{"trivy"}, Watch: []string{"trivy"}, Cache: []string{"trivy"}},
	{ID: "precommit", Name: "pre-commit", Watch: []string{"pre-commit"}, Cache: []string{"pre-commit"}},
	{ID: "golint", Name: "Go lint", Watch: []string{"golangci-lint", "go"}, Cache: []string{"golangci-lint"}},
	{ID: "thumbnails", Name: "Desktop thumbnails", Cache: []string{"thumbnails"}},
	{ID: "mise", Name: "mise installer downloads", Watch: []string{"mise"}, Cache: []string{"mise/downloads-cas"}},
}
