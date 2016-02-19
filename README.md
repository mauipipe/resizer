# resizer [![Build Status](https://travis-ci.org/hellofresh/resizer.svg?branch=master)](https://travis-ci.org/hellofresh/resizer)

This is a naive approach to build an image resizing service. At the moment given few parameters the system returns the image resized.

At the moment this service supports those versions of Go:

- 1.3
- 1.4
- latest stable version

#### How it works?

By now it listen automatically to port 8080 by default (this should be changed in the near future). 

Resizing endpoint:

GET host:8080/resize

**Parameters**:
- image: Current image url you want to change
- size: You can specify an image size like 100x100 or an already predefined placeholder like thumbnail

#### Configuration

By default we provide a dummy config.json file with some useless default values. In this configuration you can do:

- Configure default port to listen
- List of white hosts allowed to resize the image
- Max height and width for the new image
- List of placeholders with predefined size

About the hosts by default all of them are restricted. But you can add as many hosts as you want and you can use regular expressions!

For example you can do something like this:

```json
{
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

#### Dependencies

This service relies on top of some greate packages like:

- https://github.com/spf13/viper
- https://github.com/nfnt/resize

#### TODO

- [x] Resize a given image with width/height parameters
- [x] Create some unit tests
- [ ] Gopher even more this code
- [x] Configure server with configuration files
- [x] Move validators to another Go file
- [x] Allow to find hosts by regex patterns
- [x] Allow to have placeholders with default sizes
