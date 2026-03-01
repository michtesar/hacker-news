class Hnx < Formula
  desc "Hacker News design TUI"
  homepage "https://github.com/michael/hacker-news"
  license "MIT"
  head "https://github.com/michael/hacker-news.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/hnx"
  end

  test do
    assert_match "hnx", shell_output("#{bin}/hnx --help")
  end
end
