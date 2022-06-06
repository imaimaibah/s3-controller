## How to run 

```sh
./run.sh
```

## Specification

### On delete
It looks for a file named "DO_NOT_DELETE" in the root directory. If the file exists, 
it doesn't not delete the bucket when you delete corresponding k8s objects.

Process:
Check if "DO_NOT_DELETE" file exists:
  Delete k8s object only
If not
  Delete all files in bucket and delete the bucket. On successful delete, k8s object gets deleted.




