package main

import (
	"flag"
	"fmt"
	"github.com/jonmol/gphoto2"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"
)

const (
	EosRemoteRelease = "eosremoterelease"
	BatteryLevel     = "batterylevel"
)

/* CameraFiles is a list of files in CameraFilePath format */
type CameraFiles []gphoto2.CameraFilePath

/* LoadCameraFiles retrieves a list of files stored in the camera */
func (c *CameraFiles) LoadCameraFiles(camera *gphoto2.Camera) error {
	/* list files on camera */
	storage, err := camera.ListFiles()
	if err != nil {
		return err
	}
	/* walk through camera files */
	for _, device := range storage {
		for _, container := range device.Children {
			for _, directory := range container.Children {
				for _, file := range directory.Children {
					if !file.Dir {
						*c = append(*c, file)
					}
				}
			}
		}
	}
	return nil
}

/* Contains returns true if CameraFiles list contains specified file */
func (c CameraFiles) Contains(file gphoto2.CameraFilePath) bool {
	for _, f := range c {
		if f.Name == file.Name {
			return true
		}
	}
	return false
}

/* FindNew returns list of new items in files that do not exist in CameraFiles list c */
func (c *CameraFiles) FindNew(files *CameraFiles) *CameraFiles {
	result := new(CameraFiles)
	if len(*c) == len(*files) {
		return result
	}
	for _, newFile := range *files {
		if !c.Contains(newFile) {
			*result = append(*result, newFile)
		}
	}
	return result
}

/* Camera extends *gphoto2.Camera type */
type Camera struct {
	camera   *gphoto2.Camera
	Model    string
	Lens     string
	Battery  string
	ISO      int
	Aperture float64
	Shutter  string
	Duration int
	Frames   int
	Current  int
	Target   string
	Kind     string
	Keep     bool
	Files    CameraFiles
}

/* SetConfig configures integer camera setting */
func (c *Camera) SetConfig(CameraSetting string, value string) error {
	setting, err := c.camera.GetSetting(CameraSetting)
	if err != nil {
		return err
	}
	if err := setting.Set(value); err != nil {
		return err
	}
	return nil
}

/* GetBatteryStatus retrieves current battery status */
func (c *Camera) GetBatteryStatus() (level string, err error) {
	battery, err := c.camera.GetSetting(BatteryLevel)
	if err != nil {
		return "", err
	}
	v, err := battery.Get()
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

/* Status generates a real-time frame capture status */
func (c *Camera) Status(frame int, seconds int) string {
	if c.Frames == 0 {
		return fmt.Sprintf(
			"Capturing %s frame %3d; %3d seconds remaining; battery: %s",
			c.Kind,
			frame,
			seconds,
			c.Battery,
		)
	}
	return fmt.Sprintf(
		"Capturing %s frame %3d/%d; %3d seconds remaining; battery: %s",
		c.Kind,
		frame,
		c.Frames,
		seconds,
		c.Battery,
	)
}

/* CaptureBulb instructs camera to capture image with the specified duration in BULB mode */
func (c *Camera) CaptureBulb(frame int) error {
	/* get current battery status */
	battery, err := c.GetBatteryStatus()
	if err != nil {
		return err
	}
	c.Battery = battery
	/* start frame exposure */
	if err := c.SetConfig(EosRemoteRelease, "Immediate"); err != nil {
		return err
	}
	/* print loop */
	go func() {
		for left := c.Duration; left > 0; left-- {
			fmt.Printf("%s\r", c.Status(frame, left))
			time.Sleep(time.Second)
		}
	}()

	/* wait for the specified duration */
	time.Sleep(time.Second*time.Duration(c.Duration) + time.Millisecond*100)

	/* stop frame exposure */
	if err := c.SetConfig(EosRemoteRelease, "Release Full"); err != nil {
		return err
	}
	/* wait for a couple of seconds for camera to finish  */
	time.Sleep(time.Second * 2)
	/* reset camera connection */
	if err := c.camera.Reset(); err != nil {
		return err
	}
	/* get new list of files on the camera */
	files := new(CameraFiles)
	if err := files.LoadCameraFiles(c.camera); err != nil {
		return err
	}
	newFiles := c.Files.FindNew(files)
	for _, file := range *newFiles {
		/* prepare file for frame download */
		fh, err := os.Create(fmt.Sprintf("%s/%s/%s", c.Target, c.Kind, file.Name))
		if err != nil {
			return err
		}
		defer fh.Close()
		/* download frame */
		if err := file.DownloadImage(fh, false); err != nil {
			return err
		}
	}

	return nil
}

/* Close camera and free memory */
func (c *Camera) Close() error {
	if err := c.camera.Exit(); err != nil {
		return err
	}
	if err := c.camera.Free(); err != nil {
		return err
	}
	return nil
}

/* Initialize camera settings before shooting session */
//func (c *Camera) Initialize(frames uint32, duration, iso int, shutter string, aperture float64, target, kind string, keep bool) error {
func (c *Camera) Init(name string) (err error) {
	/* initialize camera parameters */
	c.camera, err = gphoto2.NewCamera(name)
	if err != nil {
		return err
	}
	/* get camera model */
	model, err := c.camera.GetSetting("cameramodel")
	if err != nil {
		return fmt.Errorf("Init(cameramodel): %v\n", err)
	}
	modelStr, err := model.Get()
	if err != nil {
		return fmt.Errorf("Init(model): %v\n", err)
	}
	c.Model = modelStr.(string)
	/* get lens name */
	lens, err := c.camera.GetSetting("lensname")
	if err != nil {
		return fmt.Errorf("Init(lensname): %v\n", err)
	}
	lensStr, err := lens.Get()
	if err != nil {
		return fmt.Errorf("Init(lens): %v\n", err)
	}
	c.Lens = lensStr.(string)
	/* perform initial camera files lookup */
	if err = c.Files.LoadCameraFiles(c.camera); err != nil {
		return
	}

	fmt.Printf("Initializing camera: %s... ", c.Model)
	if err := c.SetConfig("focusmode", "Manual"); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(focusmode): %v", err)
	}
	if err := c.SetConfig("shutterspeed", c.Shutter); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(shutterspeed): %v", err)
	}
	if err := c.SetConfig("iso", strconv.Itoa(c.ISO)); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(iso): %v", err)
	}
	if err := c.SetConfig("whitebalance", "Daylight"); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(whitebalance): %v", err)
	}
	if err := c.SetConfig("imageformat", "RAW"); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(imageformat): %v", err)
	}
	if err := c.SetConfig("aperture", strconv.FormatFloat(c.Aperture, 'f', 1, 32)); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(aperture): %v", err)
	}
	if err := c.SetConfig("capturetarget", "Memory card"); err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(capturetarget): %v\n", err)
	}
	/* get current battery status */
	battery, err := c.GetBatteryStatus()
	if err != nil {
		fmt.Printf("Error!\n")
		return fmt.Errorf("Init(batterylevel): %v\n", err)
	}
	c.Battery = battery
	fmt.Printf("Done.\n")
	return nil
}

/* CaptureLoop performs frames capture with specified parameters */
func (c *Camera) CaptureLoop() error {
	/* capture loop */
	for frame := int(0); c.Frames == 0 || frame < c.Frames; frame++ {
		/* perform frame capture */
		if err := c.CaptureBulb(frame + 1); err != nil {
			return err
		}
	}
	fmt.Printf("\n\nFrames capture complete.\n")
	return nil
}

/* main program */
func main() {
	camera := new(Camera)
	flag.IntVar(&camera.Frames, "frames", 0, "Number of images to take or 0 for no limit (default: 0)")
	flag.StringVar(&camera.Target, "target", "/tmp/target", "Name of target directory to download images to")
	flag.IntVar(&camera.Duration, "duration", 60, "Length of frames to take (default: 60s)")
	flag.StringVar(&camera.Shutter, "shutter", "bulb", "Set the specified camera shutter speed (default: 'bulb')")
	flag.Float64Var(&camera.Aperture, "aperture", 2.8, "Lens aperture ratio (default: 2.8)")
	flag.IntVar(&camera.ISO, "iso", 800, "ISO value (default: 800)")
	flag.StringVar(&camera.Kind, "kind", "lights", "Specify lights or darks frames capturing (default: lights)")
	flag.BoolVar(&camera.Keep, "keep", false, "Keep files on the camera after download (default: remove files)")
	cameraName := flag.String("name", "", "Name of camera to use (default: '')")
	flag.Parse()
	/* sanity checks */
	if camera.Kind != "lights" && camera.Kind != "darks" {
		fmt.Printf("Bad 'kind' option: %s (must be either 'lights' or 'darks'", camera.Kind)
		return
	}
	if camera.Frames*camera.Duration > 28800 {
		fmt.Printf("Specified shooting time is longer than 8 hours, aborting.\n")
		return
	}
	/* initialize camera */
	if err := camera.Init(*cameraName); err != nil {
		log.Fatal(err)
	}

	/* print camera info */
	fmt.Printf("Camera Model:  %s\n", camera.Model)
	fmt.Printf("Lens Model:    %s\n", camera.Lens)
	fmt.Printf("SD Card Files: %d\n", len(camera.Files))
	fmt.Printf("Battery Level: %s\n\n", camera.Battery)

	/* handle ctrl-c events and exit on sigint */
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			/* release button if camera is capturing a frame */
			camera.SetConfig(EosRemoteRelease, "Release Full")
			os.Exit(1)
		}
	}()

	/* Perform frames capture */
	if err := camera.CaptureLoop(); err != nil {
		log.Fatal(err)
	}

	/* close camera and free resources */
	if err := camera.Close(); err != nil {
		log.Fatal(err)
	}
}
