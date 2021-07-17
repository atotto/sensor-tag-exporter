package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/sensor-tag-exporter/influxdb"
	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
)

var (
	UUID_DEVINFO_SERV  = ble.MustParse(`0000180a-0000-1000-8000-00805f9b34fb`)
	UUID_DEVINFO_FWREV = ble.MustParse(`00002A26-0000-1000-8000-00805f9b34fb`)

	// IR_TEMPERATURE
	UUID_IR_TEMPERATURE_SERV = ble.MustParse(`f000aa00-0451-4000-b000-000000000000`)
	UUID_IR_TEMPERATURE_DATA = ble.MustParse(`f000aa01-0451-4000-b000-000000000000`)
	UUID_IR_TEMPERATURE_CONF = ble.MustParse(`f000aa02-0451-4000-b000-000000000000`) // 0: disable, 1: enable
	UUID_IR_TEMPERATURE_PERI = ble.MustParse(`f000aa03-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	// ACCELEROMETER
	UUID_ACC_SERV = ble.MustParse(`f000aa10-0451-4000-b000-000000000000`)
	UUID_ACC_DATA = ble.MustParse(`f000aa11-0451-4000-b000-000000000000`)
	UUID_ACC_CONF = ble.MustParse(`f000aa12-0451-4000-b000-000000000000`) // 0: disable, 1: enable
	UUID_ACC_PERI = ble.MustParse(`f000aa13-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	// HUMIDITY
	UUID_HUMIDITY_SERV = ble.MustParse(`f000aa20-0451-4000-b000-000000000000`)
	UUID_HUMIDITY_DATA = ble.MustParse(`f000aa21-0451-4000-b000-000000000000`)
	UUID_HUMIDITY_CONF = ble.MustParse(`f000aa22-0451-4000-b000-000000000000`) // 0: disable, 1: enable
	UUID_HUMIDITY_PERI = ble.MustParse(`f000aa23-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	// MAGNETOMETER
	UUID_MAGNETOMETER_SERV = ble.MustParse(`f000aa30-0451-4000-b000-000000000000`)
	UUID_MAGNETOMETER_DATA = ble.MustParse(`f000aa31-0451-4000-b000-000000000000`)
	UUID_MAGNETOMETER_CONF = ble.MustParse(`f000aa32-0451-4000-b000-000000000000`) // 0: disable, 1: enable
	UUID_MAGNETOMETER_PERI = ble.MustParse(`f000aa33-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	// OPTICAL
	UUID_OPTICAL_SERV = ble.MustParse(`f000aa70-0451-4000-b000-000000000000`)
	UUID_OPTICAL_DATA = ble.MustParse(`f000aa71-0451-4000-b000-000000000000`)
	UUID_OPTICAL_CONF = ble.MustParse(`f000aa72-0451-4000-b000-000000000000`) // 0: disable, 1: enable
	UUID_OPTICAL_PERI = ble.MustParse(`f000aa73-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	// BAROMETER
	UUID_BAROMETER_SERV = ble.MustParse(`f000aa40-0451-4000-b000-000000000000`)
	UUID_BAROMETER_DATA = ble.MustParse(`f000aa41-0451-4000-b000-000000000000`)
	UUID_BAROMETER_CONF = ble.MustParse(`f000aa42-0451-4000-b000-000000000000`) // 0: disable, 1: enable
	UUID_BAROMETER_CALI = ble.MustParse(`f000aa43-0451-4000-b000-000000000000`) // Calibration characteristic
	UUID_BAROMETER_PERI = ble.MustParse(`f000aa44-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	// GYROSCOPE
	UUID_GYROSCOPE_SERV = ble.MustParse(`f000aa50-0451-4000-b000-000000000000`)
	UUID_GYROSCOPE_DATA = ble.MustParse(`f000aa51-0451-4000-b000-000000000000`)
	UUID_GYROSCOPE_CONF = ble.MustParse(`f000aa52-0451-4000-b000-000000000000`) // 0: disable, bit 0: enable x, bit 1: enable y, bit 2: enable z
	UUID_GYROSCOPE_PERI = ble.MustParse(`f000aa53-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	UUID_MOVEMENT_SERV = ble.MustParse(`f000aa80-0451-4000-b000-000000000000`)
	UUID_MOVEMENT_DATA = ble.MustParse(`f000aa81-0451-4000-b000-000000000000`)
	UUID_MOVEMENT_CONF = ble.MustParse(`f000aa82-0451-4000-b000-000000000000`) // 0: disable, bit 0: enable x, bit 1: enable y, bit 2: enable z
	UUID_MOVEMENT_PERI = ble.MustParse(`f000aa83-0451-4000-b000-000000000000`) // Period in tens of milliseconds

	UUID_TST_SERV = ble.MustParse(`f000aa64-0451-4000-b000-000000000000`)
	UUID_TST_DATA = ble.MustParse(`f000aa65-0451-4000-b000-000000000000`) // Test result

	UUID_KEY_SERV = ble.MustParse(`0000ffe0-0000-1000-8000-00805f9b34fb`)
	UUID_KEY_DATA = ble.MustParse(`0000ffe1-0000-1000-8000-00805f9b34fb`)
)

var influxdbURL = os.Getenv("SENSORTAG_INFLUXDB_URL")

func main() {
	flag.Parse()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sig
		cancel()
	}()

	device, err := dev.DefaultDevice()
	if err != nil {
		log.Fatal(err)
	}

	ble.SetDefaultDevice(device)

	go func() {
		log.Println("connecting...")

		client, err := ble.Connect(ctx, func(a ble.Advertisement) bool {
			if a.Connectable() && strings.Contains(a.LocalName(), "CC2650 SensorTag") {
				log.Printf("connect to %s", a.LocalName())
				return true
			}
			return false
		})
		if err != nil {
			log.Fatalf("failed to connect: %s", err)
		}
		go func() {
			<-client.Disconnected()
			cancel()
		}()

		p, err := client.DiscoverProfile(true)
		if err != nil {
			log.Fatalf("failed to discover profile: %s", err)
		}

		//enableServices := []ble.UUID{UUID_IR_TEMPERATURE_SERV}
		//for _, svc := range enableServices {
		//
		//}

		irTempSvc := p.FindService(ble.NewService(UUID_IR_TEMPERATURE_SERV))
		humiditySvc := p.FindService(ble.NewService(UUID_HUMIDITY_SERV))
		barometerSvc := p.FindService(ble.NewService(UUID_BAROMETER_SERV))
		opticSvc := p.FindService(ble.NewService(UUID_OPTICAL_SERV))

		sensorChars := []*ble.Characteristic{
			irTempSvc.Characteristics[1],
			humiditySvc.Characteristics[1],
			barometerSvc.Characteristics[1],
			opticSvc.Characteristics[1],
		}

		tc := time.Tick(time.Minute)
		var num uint64
		for {
			fields := make([]string, 0, 16)
			// enable
			for _, c := range sensorChars {
				if err := client.WriteCharacteristic(c, []byte{0x01}, true); err != nil {
					log.Println("enable: %s", err)
				}
			}

			{
				data, err := client.ReadCharacteristic(irTempSvc.Characteristics[0])
				if err != nil {
					log.Println("read data: %s", err)
					cancel()
					return
				}

				tdie := float64(uint16(data[2])|uint16(data[3])<<8>>2) * 0.03125
				tObj := float64(uint16(data[0])|uint16(data[1])<<8>>2) * 0.03125
				fields = append(fields, fmt.Sprintf("ir_die_temp=%.2f", tdie))
				fields = append(fields, fmt.Sprintf("ir_obj_temp=%.2f", tObj))
				log.Printf("die temp %.2f object temp %.2f", tdie, tObj)
			}
			{
				data, err := client.ReadCharacteristic(humiditySvc.Characteristics[0])
				if err != nil {
					log.Println("read data: %s", err)
					cancel()
					return
				}

				temp := (float64(uint16(data[0])|uint16(data[1])<<8)/65536)*165 - 40
				hum := float64((uint16(data[2])|uint16(data[3])<<8) & ^uint16(0x0003)) / 65536 * 100
				fields = append(fields, fmt.Sprintf("temperature=%.2f", temp))
				fields = append(fields, fmt.Sprintf("humidity=%.2f", hum))
				log.Printf("temperature %.2f humidity %.2f", temp, hum)
			}
			{
				data, err := client.ReadCharacteristic(barometerSvc.Characteristics[0])
				if err != nil {
					log.Println("read data: %s", err)
					cancel()
					return
				}

				temp := float64(uint32(data[0])|uint32(data[1])<<8|uint32(data[2])<<16) / 100.0
				press := float64(uint32(data[3])|uint32(data[4])<<8|uint32(data[5])<<16) / 100.0
				fields = append(fields, fmt.Sprintf("temperature2=%.2f", temp))
				fields = append(fields, fmt.Sprintf("pressure=%.2f", press))
				log.Printf("temperature %.2f pressure %.2f", temp, press)
			}
			{
				data, err := client.ReadCharacteristic(opticSvc.Characteristics[0])
				if err != nil {
					log.Println("read data: %s", err)
					cancel()
					return
				}

				rawData := uint16(data[0]) | uint16(data[1])<<8
				m := rawData & 0x0FFF
				e := (rawData & 0xF000) >> 12
				if e == 0 {
					e = 1
				} else {
					e = 2 << (e - 1)
				}
				light := float64(m) * (0.01 * float64(e))
				fields = append(fields, fmt.Sprintf("illuminance=%.2f", light))
				log.Printf("illuminance %.2f", light)
			}

			// disable
			for _, c := range sensorChars {
				if err := client.WriteCharacteristic(c, []byte{0x01}, true); err != nil {
					log.Println("enable: %s", err)
				}
			}
			if num != 0 {
				line := bytes.NewBuffer(nil)
				if err := influxdb.WriteLineProtocol(line, "sensortag", nil, fields, time.Now()); err != nil {
					log.Fatal(err)
				}
				os.Stdout.Write(line.Bytes())
				if err := influxdb.PostBuffer(influxdbURL, line); err != nil {
					log.Fatal(err)
				}
			}
			num++

			select {
			case <-ctx.Done():
				return
			case <-client.Disconnected():
				return
			case <-tc:
			}
		}
	}()

	<-ctx.Done()

	log.Println("done")
}
