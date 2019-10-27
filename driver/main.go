package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/gousb"
)

func main() {
	stage1 := flag.Bool("stage1", false, "USB mode")
	stage2 := flag.Bool("stage2", false, "FPGA mode and run time")
	flag.Parse()

	ctx := gousb.NewContext()
	defer ctx.Close()

	log.Printf("Opening device...")

	dev, err := ctx.OpenDeviceWithVIDPID(0x5555, 0x3382)
	if err != nil {
		log.Fatalf("Could not open a device: %v", err)
	}
	defer dev.Close()

	log.Printf("Claim default interface...")

	deviceConfig, err := dev.Config(1)
	if err != nil {
		log.Printf("Failed to get config for device")
		return
	}

	for num, interf := range deviceConfig.Desc.Interfaces {
		fmt.Printf("\tINTERFACE #%d\n", num)

		for numAlt, altset := range interf.AltSettings {
			fmt.Printf("\t\t%d - Number: %d - Alt %d\n", numAlt, altset.Number, altset.Alternate)
			for numEndpoint, endpointset := range altset.Endpoints {
				fmt.Printf("\t\t\t%d - %#v\n", numEndpoint, endpointset)
			}
		}
	}

	// dev.ControlTimeout = time.Second
	if *stage1 {
		log.Printf("Setting the USB Controller firmware I think...")

		for n, packet := range usbInit {
			if n%10 == 0 {
				thrd := len(usbInit) / 3
				switch n / thrd {
				case 0:
					fmt.Print("U")
				case 1:
					fmt.Print("S")
				case 2:
					fmt.Print("B")
				}
			}

			_, err = dev.Control(packet.RequestType, packet.BRequest, packet.WValue, packet.WIndex, packet.Data)
			if err != nil {
				log.Printf("\nfailed to bitstream %s", err.Error())
			}
		}

		time.Sleep(time.Second)
		os.Exit(0)
	} else if *stage2 {
		buf := make([]byte, 64)
		_, err = dev.Control(0xc0, 185, 0x0021, 0, buf)

		log.Printf("Setting the FPGA bitstream I think...")

		for n, packet := range fpgaInit {
			if n%60 == 0 {
				thrd := len(fpgaInit) / 4
				switch n / thrd {
				case 0:
					fmt.Print("F")
				case 1:
					fmt.Print("P")
				case 2:
					fmt.Print("G")
				case 3:
					fmt.Print("A")
				}
			}

			_, err = dev.Control(packet.RequestType, packet.BRequest, packet.WValue, packet.WIndex, packet.Data)
			if err != nil {
				log.Printf("\nfailed to bitstream %s", err.Error())
			}
		}

	} else {
		log.Fatalf("Set a -stage1 or -stage2")
	}

	log.Printf("Activating in 2 seconds")
	time.Sleep(time.Second * 2)

	//  unable to grab default interface:
	// vid=5555,pid=3382,bus=3,addr=63,config=1,if=0,alt=0
	// does not have endpoint with address 0x80.
	// Available endpoints: [0x82(2,IN)]

	cfg, err := dev.Config(1)
	if err != nil {
		log.Fatalf("Kaboom, unable to grab Config(1): %s", err.Error())
	}
	fmt.Print("Config: ")

	defaultinterface, err := cfg.Interface(0, 0)
	if err != nil {
		log.Fatalf("Kaboom, unable to grab Interface(0, 0): %s", err.Error())
	}
	fmt.Print("Interface: ")

	// defaultInterface, done, err := dev.DefaultInterface()
	// defer done()
	// if err != nil {
	// 	log.Fatalf("Kaboom, unable to grab default interface: %s", err.Error())
	// }

	inputTest, err := defaultinterface.InEndpoint(2)
	if err != nil {
		log.Fatalf("Kaboom, unable to grab default interface: %s", err.Error())
	}

	fmt.Print("Endpoint: ")

	buf := make([]byte, 39)
	_, err = dev.Control(0x40, 176, 0, 0, buf)
	fmt.Print("Null 39 Control: ")

	inBuf := make([]byte, 12)
	_, err = dev.Control(0xc0, 177, 0, 0, inBuf)
	fmt.Print("Grab In 12 Control: \n")

	log.Printf("Here is a blob that I don't know what it does %x", inBuf)

	// now to send what i think activates this

	go func() {

		for {
			dev.Control(0x40, 176, 0, 0, []byte("\x04\x10\x00\x00\x10\x14\x10\xaa\xaa\xaa\x1f\x1f\x1f\x03\x03\x20\x02\x58\x00\x01\x02\x58\x00\x7c\x19\x01\x01\x80\x80\x00\x00\x00\x00\x00\x00\x03\x20\x02\x58"))
			time.Sleep(time.Millisecond * 200)
			dev.Control(0x40, 184, 0x0076, 0, []byte(""))
		}

	}()

	f, _ := os.Create("./debug")

	bytesRead := 0
	for {
		dataBuf := make([]byte, 61440*6)
		n, err := inputTest.Read(dataBuf)
		if err != nil {
			log.Printf("failed to read aa %s", err)
			continue
		}

		bytesRead += n

		f.Write(dataBuf[:n])
		fmt.Printf("\rBytes Read: %d   ", bytesRead)
	}

	time.Sleep(time.Hour)

}

type usbControlPacket struct {
	ignore      bool
	RequestType uint8
	BRequest    uint8
	WValue      uint16
	WIndex      uint16
	Data        []byte
}
