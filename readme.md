1) Pick any file on a public web server over say 60MB, either a file you upload or a preexisting file somewhere.
2) Using Go, write a multi-source downloader for that single file. In other words, a downloader that downloads different chunks of the file over HTTP or HTTPS and reassembles the chunks on the client side into the complete file. It should download chunks in parallel. The idea is that in theory this would work if different chunks of the file were actually hosted on different servers, a la BitTorrent or the many things that did this before BitTorrent. The client also should not be expected to know the size of the file ahead of time.
3) If the server host happens to return an etag in a known hash format, use that hash format to verify that the downloaded file is correct. This is not a requirement.
4) Make it so that we can easily run the downloader.

Example with ranges:
https://getsamplefiles.com/download/mp4/sample-1.mp4

Default with no ranges:
https://filesamples.com/samples/video/mp4/sample_1280x720_surfing_with_audio.mp4