# Homebrew packaging

`Formula/lazycont.rb` is a tap-ready Homebrew formula for installing lazycont from source.

The formula is HEAD-only until the project has a tagged release archive and checksum. Homebrew also requires formulae to live in a tap, so the first public packaging target should be a small tap such as `pz/homebrew-lazycont`.

## Formula smoke

After the tap is published, install the formula from the tap:

```sh
brew install --HEAD pz/lazycont/lazycont
lazycont --version
```

The formula intentionally tests `--version` and `--help` so Homebrew can verify the binary without requiring Apple's `container` CLI to be installed and initialized.

## Publish a tap

Create a tap repo named `homebrew-lazycont`, copy the formula into it, and install from the tap:

```sh
brew tap-new pz/lazycont
cp Formula/lazycont.rb "$(brew --repository pz/lazycont)/Formula/lazycont.rb"
cd "$(brew --repository pz/lazycont)"
brew style Formula/lazycont.rb
git add Formula/lazycont.rb
git commit -m "Add lazycont formula"
git remote add origin git@github.com:pz/homebrew-lazycont.git
git push -u origin main
brew install --HEAD pz/lazycont/lazycont
brew test pz/lazycont/lazycont
```

## Add a stable release

Once a first release is tagged, add a stable source archive to the formula:

```ruby
url "https://github.com/pz/lazycont/archive/refs/tags/v0.1.0.tar.gz"
sha256 "<tarball-sha256>"
```

Then rerun:

```sh
brew audit --strict --new --formula --tap pz/lazycont lazycont
brew install --build-from-source pz/lazycont/lazycont
brew test pz/lazycont/lazycont
```

## Release checklist

- choose and add a project license before submitting to a public package index
- tag a release and create a GitHub release archive
- add `url` and `sha256` to the formula
- run `brew style Formula/lazycont.rb` inside the tap
- run `brew audit --strict --new --formula --tap pz/lazycont lazycont`
- run `brew test pz/lazycont/lazycont`
