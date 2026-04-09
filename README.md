# Auth-TUI

<br>

---

<br>

### Compile:

```
make build
```

exec file will be compiled in `./built` folder.

<br>

```sh
cd built
echo "export PATH=\"\$PATH:$(pwd)\"" >> ~/.$(basename $SHELL)rc && source ~/.$(basename $SHELL)rc
```