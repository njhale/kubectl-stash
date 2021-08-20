# kubectl-stash

kubectl-stash is a kubectl plugin that lets you treat your cluster like a key-value store.

## Examples

Stash the content of a file:

```sh
$ tree
.
└── doge.svg

0 directories, 1 file
$ cat doge.svg | kubectl stash
kgggggg7
```

List stashed items:

```sh
$ kubectl stash ls
kgggggg7
```

Get a stashed item and store it in a file:

```sh
$ kubectl stash get kgggggg7 -o doge.svg
```

Remove a stashed file:

```sh
$ kubectl stash rm kgggggg7
```

Stash a directory:

```sh
$ tar -czvf - . | kubectl stash
i8sdd155
```

Unpack a stashed directory:

```sh
$ kubectl stash get i8sdd155 | tar -xzvf -
```

