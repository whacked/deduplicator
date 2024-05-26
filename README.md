# deduplicator

a simple file deduplicator for a particular use case: you have files across 2 locally-accessible locations with mostly the same hierarchical structure, say a `reference` directory and a `target` directory; you want to remove all files within `target` that have identically-pathed and identically-hashed files already existing in `reference`.

Programs like `fdupes` do not care about hierarchical location, so it will flag e.g. `__init__.py` and other often zero-byte files as dupes.

This program is mostly useful for verifying and clearing backups, when the taget directory is expected to have a high degree of similarity with the reference directory, but you want to be extra sure. In this case you can:

```
# scan for files in local directory and output the paths and hashes to a reference file on the remote
[local]$ deduplicator -refDir my/files | ssh user@backup.host 'cat > /tmp/ref.yml'
[local]$ ssh user@backup.host
[remote]$ deduplicator -refYaml /tmp/ref.yml -targetDir /backup/my/files -deleteFiles
```

on the remote, the program will read the relpaths and hashes, compare them to the yaml file, confirm, then delete the duplicate files. If you abort the deletion, it will print a "deletion plan", which is all the `rm` statements you can use to manually delete the dupes.
