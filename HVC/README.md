

```bash
g++ -std=c++17 -o homomorphic_example homomorphic_example.cpp -I/usr/local/include/SEAL-4.1 -L/usr/local/lib -lseal-4.1

g++ -shared -o libhomomorphic_lib.so -fPIC homomorphic_example.cpp -I/usr/local/include/SEAL-4.1 -L/usr/local/lib -lseal-4.1 -std=c++17

```



```bash
go build
```