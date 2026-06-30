class Lazycontainer < Formula
  desc "Lazydocker-style terminal UI for Apple's container CLI"
  homepage "https://github.com/pzep1/lazycontainer"
  url "https://github.com/pzep1/lazycontainer/archive/refs/tags/v0.5.2.tar.gz"
  sha256 "6369fa5237e301fb8ed21fb06cd70772213b426677001883a06dc5cac7382e0e"
  license "GPL-3.0-or-later"
  head "https://github.com/pzep1/lazycontainer.git", branch: "main"

  depends_on "go" => :build
  depends_on "container"
  depends_on :macos

  def install
    build_version = version.to_s
    build_version = "HEAD" if build_version.empty?

    system "go", "build",
      *std_go_args(ldflags: "-s -w -X main.version=#{build_version}"),
      "./cmd/lazycontainer"
  end

  def caveats
    <<~EOS
      lazycontainer drives Apple's container CLI. Homebrew installs it as a
      dependency, but you still need to start its system service before
      launching the TUI:

        brew services start container

      Or start it manually for the current session:

        container system start
    EOS
  end

  test do
    assert_match "lazycontainer", shell_output("#{bin}/lazycontainer --version")
    assert_match "Usage:", shell_output("#{bin}/lazycontainer --help")
  end
end
