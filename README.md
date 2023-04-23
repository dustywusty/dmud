# ...

```
find internal -type f -name '*.go' -exec sh -c 'echo "=== {} ==="; cat {}' \;
./scripts/build.sh && ./dmud
```