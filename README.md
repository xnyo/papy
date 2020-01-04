# 🎷 Papy
Papy is an (incremental) build system from Skyrim Special Edition mods, written in Go.

## Features
- [x] Incremental scripts compile system
- [ ] BSA packing
- [ ] BSA splitting
- [ ] BSA files aggregation

## How to use it
Right now, Papy is only able to compile scripts. In the future, it'll be able to pack, split and aggregate files inside BSA archives (like [pigroman](https://github.com/xnyo/pigroman) does). However, Papy will compile only scripts that have to be re-compiled, by comparing the psc and pex timestamps. Here's how you use it:

Get papy:

```bash
go get -v -u github.com/xnyo/papy
```

(assuming `$GOPATH/bin` or `$GOBIN` is in your PATH)

Create the global config file (it will detect/prompt for your papyrus compiler, you need to run this only once)
```
papy setup
```

Put a file called `papy.yaml` in your project root (usually `ModOrganizer\mods\yourmod`), and populate it:

```yaml
output_folders:
  - scripts
optimize: false
imports:
  - $base_game
folders:
  - source\scripts
```

Then run `papy incremental` in your project root to compile the scripts that have been modified.

## Licence
MIT
