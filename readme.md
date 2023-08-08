# Instructions
1) Pick any file on a public web server over say 60MB, either a file you upload or a preexisting file somewhere.
2) Using Go, write a multi-source downloader for that single file. In other words, a downloader that downloads different chunks of the file over HTTP or HTTPS and reassembles the chunks on the client side into the complete file. It should download chunks in parallel. The idea is that in theory this would work if different chunks of the file were actually hosted on different servers, a la BitTorrent or the many things that did this before BitTorrent. The client also should not be expected to know the size of the file ahead of time.
3) If the server host happens to return an etag in a known hash format, use that hash format to verify that the downloaded file is correct. This is not a requirement.
4) Make it so that we can easily run the downloader.

# Application
The application is a simple multi-source downloader that downloads a file from different sources and reassembles the chunks on the client side into the complete file. It downloads chunks in parallel.
If the server host happens to know file size and supports range header, it will use that to download file in parallel. Otherwise, it will download file in dumb alternative mode where it will spown multiple go routines to download file in parallel.

# How to run
The application is written in go. To run the application, you need to have go installed on your machine.

To run the application, run the following command from the root directory of the project.
```
go run main.go -urlsFile=parallelURLs.txt
```

# Help
``` 
go run main.go -help
Usage:
  -chunkPrefix string
        Prefix of the chunk files (default "chunk-")
  -chunkSize int
        Size of each chunk in bytes (default 10485760)
  -forceAlternative
        Force the alternative downloader to be used
  -numWorkers int
        Number of workers to use (default 5)
  -urlsFile string
        File containing the urls to download  