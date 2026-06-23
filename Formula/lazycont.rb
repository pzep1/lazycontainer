class Lazycont < Formula
  desc "Lazydocker-style terminal UI for Apple's container CLI"
  homepage "https://github.com/pz/lazycont"
  head "https://github.com/pz/lazycont.git", branch: "main"

  depends_on "go" => :build
  depends_on "container"
  depends_on :macos

  def install
    build_version = version.to_s
    build_version = "HEAD" if build_version.empty?

    system "go", "build",
      *std_go_args(ldflags: "-s -w -X main.version=#{build_version}"),
      "./cmd/lazycont"
  end

  def caveats
    <<~EOS
      lazycont drives Apple's container CLI. Homebrew installs it as a
      dependency, but you still need to start its system service before
      launching the TUI:

        container system start
    EOS
  end

  test do
    assert_match "lazycont", shell_output("#{bin}/lazycont --version")
    assert_match "Usage:", shell_output("#{bin}/lazycont --help")
  end
end
