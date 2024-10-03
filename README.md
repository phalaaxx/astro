[![GoDoc](https://godoc.org/github.com/phalaaxx/astro?status.svg)](https://godoc.org/github.com/phalaaxx/astro)

# astro
A Linux CLI tool based on gphoto2 library for setting up Canon camera and taking pictures for astronomical imaging purposes.

## note
The only option that does not trigger failure during initial camera setup is aperture, because not all camera lenses have
electrical connection with the camera and aperture may not always be controlled by the camera (for manual lenses).

# usage
In order to properly take pictures from the camera, it's necessary to point -target option to a directory which contains
a set of subdirectories (at least lights and darks). This is a requirement because -keep option accepts either "lights" or
"darks" option and after images are downloaded from the camera they are saved in their proper location.

	Usage of astro:
  -aperture float
        Lens aperture ratio (default: 2.8) (default 2.8)
  -duration int
        Length of frames to take (default: 60s) (default 60)
  -frames int
        Number of images to take or 0 for no limit (default: 0)
  -iso int
        ISO value (default: 800) (default 800)
  -keep
        Keep files on the camera after download (default: remove files)
  -kind string
        Specify lights or darks frames capturing (default: lights) (default "lights")
  -name string
        Name of camera to use (default: '')
  -shutter string
        Set the specified camera shutter speed (default: 'bulb') (default "bulb")
  -target string
        Name of target directory to download images to (default "/tmp/target")


## examples

Make necessary subdirectories in the target tree:

	mkdir -p /home/user/DSO/{lights,darks,flats,biases}

Take 120 frames with 60 seconds each, aperture of 5.6 (images from the camera will be downloaded in /home/user/DSO/lights directory):

	astro -duration=60 -frames=120 -iso=1600 -kind=lights -target=/home/user/DSO

Take 30 dark frames, 60 seconds each (images from camera will be downloaded in /home/user/DSO/darks directory):

	astro -duration=60 -frames=30 -iso=1600 -kind=darks -target=/home/user/DSO
