class Odh < Formula
  desc "Agent-friendly CLI for public Open Data Hub APIs"
  homepage "https://github.com/galjos/odh-cli"
  url "https://github.com/galjos/odh-cli/archive/refs/tags/v0.2.0.tar.gz"
  sha256 "ae53b13dd7736daf4a3456acf4b611de608617241e41c4425a2957acc9610e8a"
  license "MPL-2.0"
  head "https://github.com/galjos/odh-cli.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %w[
      -s -w
      -X github.com/galjos/odh-cli/internal/version.Version=0.2.0
      -X github.com/galjos/odh-cli/internal/version.Commit=e4b74312e668
      -X github.com/galjos/odh-cli/internal/version.Date=2026-05-26T09:16:34Z
    ]

    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/odh"
  end

  test do
    assert_match "odh 0.2.0", shell_output("#{bin}/odh version --format text")
    system bin/"odh", "doctor", "--network=false"
  end
end
