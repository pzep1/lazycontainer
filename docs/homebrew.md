# Homebrew packaging

`Formula/lazycontainer.rb` is a tap-ready Homebrew formula for installing lazycontainer from source.

The formula is HEAD-only until the project has a tagged release archive and checksum. Homebrew core now ships Apple's `container` CLI as the `container` formula, so lazycontainer can depend on that package instead of asking users to install the CLI separately. Homebrew also requires formulae to live in a tap, so the first public packaging target should be a small tap such as `pzep1/homebrew-lazycont`.

## Formula smoke

After the tap is published, install the formula from the tap:

```sh
brew install pzep1/lazycont/lazycontainer
lazycontainer --version
```

The formula intentionally tests `--version` and `--help` so Homebrew can verify the binary without requiring Apple's container service to be running.

The formula declares `depends_on "container"`, but users still need to start Apple's container service before using the TUI:

```sh
brew services start container
```

Or start it manually for the current session:

```sh
container system start
```

## Publish a tap

Create a tap repo named `homebrew-lazycont`, copy the formula into it, and install from the tap:

```sh
brew tap-new pzep1/lazycont
cp Formula/lazycontainer.rb "$(brew --repository pzep1/lazycont)/Formula/lazycontainer.rb"
cd "$(brew --repository pzep1/lazycont)"
brew style Formula/lazycontainer.rb
git add Formula/lazycontainer.rb
git commit -m "Add lazycontainer formula"
git remote add origin git@github.com:pzep1/homebrew-lazycont.git
git push -u origin main
brew install pzep1/lazycont/lazycontainer
brew test pzep1/lazycont/lazycontainer
```

## Add a stable release

Once a first release is tagged, add a stable source archive to the formula:

```ruby
url "https://github.com/pzep1/lazycont/archive/refs/tags/v0.1.0.tar.gz"
sha256 "<tarball-sha256>"
```

Then rerun:

```sh
brew audit --strict --new --formula --tap pzep1/lazycont lazycontainer
brew install --build-from-source pzep1/lazycont/lazycontainer
brew test pzep1/lazycont/lazycontainer
```

## Release checklist

- choose and add a project license before submitting to a public package index
- tag a release and create a GitHub release archive
- add `url` and `sha256` to the formula
- run `brew style Formula/lazycontainer.rb` inside the tap
- run `brew audit --strict --new --formula --tap pzep1/lazycont lazycontainer`
- run `brew test pzep1/lazycont/lazycontainer`
