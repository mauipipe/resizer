# resizer [![Build Status](https://travis-ci.org/hellofresh/resizer.svg?branch=master)](https://travis-ci.org/hellofresh/resizer) [![Docker Repository on Quay](https://quay.io/repository/hellofresh/resizer/status "Docker Repository on Quay")](https://quay.io/repository/hellofresh/resizer)

#### What is this?

This is a microservice to help you to resize images on the fly. This can been built in order to scale and support a high load traffic and peaks.

#### Architecture

![Architecture graph](http://i.imgur.com/aNxYVP2.png)

This graph basically represents our infrastacture in order to store images.

Every time we request a new image it's done first of all through Amazon CloudFront. If ACF already has a copy of that image then we are done.

If ACF doesn't have a copy then the Resizer service is called. This service will try to find the image first of all in it's own cache layer (at the moment 1GB of disk space). If the image exists there it's resized and deliver to ACF.

If the image doesn't exists on it's cache layer the image is downloaded from Amazon S3. After that the image is resized and we store the original image size to the cache.

In this way we are reducing the amount of calls to Amazon and we can deliver images between 0.1 and 1 second (depending on load and size).

#### Configuration

By default we provide a dummy config.json file with some useless default values. In this configuration you can do:

- Configure default port to listen
- Configure the path to the aws bucket
- List of white hosts allowed to resize the image
- Max height and width for the new image
- List of placeholders with predefined size

About the hosts by default all of them are restricted. But you can add as many hosts as you want and you can use regular expressions!

For example you can do something like this:

```json
{
    "imagehost": "https://amazon.s3.bucket.com",
    "hostwhitelist": [
        "([a-z]+).supercdn.com"
    ],
    "sizelimits": {
        "height": 1000,
        "width": 1000
    },
    "placeholders" : [
        {
            "name": "thumbnail",
            "size": {
                "width": 100,
                "height": 100
            }
        }
    ]
}
```

The previous example show you how to allow any something.supercdn.com host.

#### Endpoints

##### Health Status & Stats

This endpoints returns a 200 http code and a json payload if everything is alright. The payload looks like this:

```json
{
	"status": "ok",
	"cache": [{
		"file_cache": {
			"hits": 193,
			"misses": 96
		}
	}, {
		"lru_cache": {
			"hits": 5382,
			"misses": 40,
			"size": 38
		}
	}],
	"used_space": "4.017240 Mb"
}
```

#### Testing

At the moment this service lacks lot of tests in many places.

##### Load testing

We used [Siege](https://www.joedog.org/siege-home/) as a tool to test the performance of this service.

#### Dependencies

This service relies on top of some great packages like:

- https://github.com/spf13/viper
- https://github.com/nfnt/resize
- https://github.com/gorilla/mux

#### Run with Docker

		$ export PRIVATE_IP=$(/sbin/ifconfig eth0 | grep 'inet addr' | cut -d: -f2 | awk '{print $1}')
		$ sudo docker-compose pull && sudo docker-compose up -d
		$ sudo docker logs -f resizer

#### TODO

- [x] Resize a given image with width/height parameters
- [x] Create some unit tests
- [ ] Gopher even more this code
- [x] Configure server with configuration files
- [x] Move validators to another Go file
- [x] Allow to find hosts by regex patterns
- [x] Allow to have placeholders with default sizes
