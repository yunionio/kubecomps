## For MacOS

```bash
brew tap mulbc/ceph-client
brew install ceph-client
```

```bash
export CGO_CPPFLAGS="-I/opt/homebrew/Cellar/ceph-client/17.2.5_1.reinstall/include"
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/ceph-client//17.2.5_1.reinstall/lib"

make cmd/kubeserver
```
