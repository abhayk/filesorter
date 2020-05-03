# filesorter
A tool to copy files to a destination folder and sort them into folders based on their modified time. I preimarily use this to copy files from the camera folder of my phone to backup hard drive. See the README for more details.

For eg a file named abc.txt last modified at May 2 2020, will end up with the following path at the destination - <destination folder>/2020/May/2/abc.txt

#### Usage
```
Usage: filesorter <source path> <destination path> [file types]
  -destination string
        The destination to which the files should be copied and sorted.
  -source string
        The source directory path,
  -types string
        Optional. Provide the list of file types that should be included from
                the source directory separated by a ':'. For eg: jpg:jpeg:mp4
```