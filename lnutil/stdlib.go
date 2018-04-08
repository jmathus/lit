package lnutil

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"
	"strconv"
)

// I shouldn't even have to write these...

// time.Time to 20 bytes. Always works.
func TimetB(givenTime time.Time) []byte {
	givenTimeString := givenTime.String()

  timeRune := []rune(givenTimeString)
  tempRune1 := append(timeRune[0:4],timeRune[5:7]...)
  tempRune2 := append(tempRune1[:],timeRune[8:10]...)
  tempRune3 := append(tempRune2[:],timeRune[11:13]...)
  tempRune4 := append(tempRune3[:],timeRune[14:16]...)
  tempRune5 := append(tempRune4[:],timeRune[17:19]...)
  tempRune6 := append(tempRune5[:],timeRune[20:26]...)
  timeBytes := []byte(string(tempRune6))

  return timeBytes
}

// bytes to time.Time. Always works.
func BtTime(BArr []byte) time.Time {
	newTimeString := string(BArr[:])
  newTimeRune := []rune(newTimeString)

  year, err := strconv.Atoi(string(newTimeRune[0:4]))
  if err != nil {
    fmt.Println(err)
  }
  month, err := strconv.Atoi(string(newTimeRune[4:6]))
  if err != nil {
    fmt.Println(err)
  }
  day, err := strconv.Atoi(string(newTimeRune[6:8]))
  if err != nil {
    fmt.Println(err)
  }
  hour, err := strconv.Atoi(string(newTimeRune[8:10]))
  if err != nil {
    fmt.Println(err)
  }
  minute, err := strconv.Atoi(string(newTimeRune[10:12]))
  if err != nil {
    fmt.Println(err)
  }
  second, err := strconv.Atoi(string(newTimeRune[12:14]))
  if err != nil {
    fmt.Println(err)
  }
  nanosecond, err := strconv.Atoi(string(newTimeRune[14:20]))
  if err != nil {
    fmt.Println(err)
  }
  location := time.Local
  byteTime := time.Date(year, time.Month(month) , day, hour, minute, second, nanosecond, location)

	return byteTime
}

// []int32 to 20 bytes. Always works.
func I32ArrtB(intArr []int32) []byte {
  var byteArr []byte

  for i := range intArr {
    var buf bytes.Buffer
  	binary.Write(&buf, binary.BigEndian, intArr[i])
    byteArr = append(byteArr, buf.Bytes()[3])
  }

  return byteArr
}

// bytes to []int32. Always works.
func BtI32Arr(BArr []byte) []int32 {
  var dataConverted []int32

  for i := range BArr {
    dataConverted = append(dataConverted, int32(uint32(BArr[i])))
  }

  return dataConverted
}

// int32 to 4 bytes.  Always works.
func I32tB(i int32) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// uint32 to 4 bytes.  Always works.
func U32tB(i uint32) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// 4 byte slice to uint32.  Returns ffffffff if something doesn't work.
func BtU32(b []byte) uint32 {
	if len(b) != 4 {
		fmt.Printf("Got %x to BtU32 (%d bytes)\n", b, len(b))
		return 0xffffffff
	}
	var i uint32
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// 4 byte slice to int32.  Returns 7fffffff if something doesn't work.
func BtI32(b []byte) int32 {
	if len(b) != 4 {
		fmt.Printf("Got %x to BtI32 (%d bytes)\n", b, len(b))
		return 0x7fffffff
	}
	var i int32
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// uint64 to 8 bytes.  Always works.
func U64tB(i uint64) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// int64 to 8 bytes.  Always works.
func I64tB(i int64) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, i)
	return buf.Bytes()
}

// 8 bytes to int64 (bitcoin amounts).  returns 7fff... if it doesn't work.
func BtI64(b []byte) int64 {
	if len(b) != 8 {
		fmt.Printf("Got %x to BtI64 (%d bytes)\n", b, len(b))
		return 0x7fffffffffffffff
	}
	var i int64
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// 8 bytes to uint64.  returns ffff. if it doesn't work.
func BtU64(b []byte) uint64 {
	if len(b) != 8 {
		fmt.Printf("Got %x to BtU64 (%d bytes)\n", b, len(b))
		return 0xffffffffffffffff
	}
	var i uint64
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &i)
	return i
}

// NopeString returns true if the string means "nope"
func NopeString(s string) bool {
	nopes := []string{
		"nope", "no", "n", "false", "0", "nil", "null", "disable", "off", "",
	}
	for _, ts := range nopes {
		if ts == s {
			return true
		}
	}
	return false
}

// YupString returns true if the string means "yup"
func YupString(s string) bool {
	yups := []string{
		"yup", "yes", "y", "true", "1", "ok", "enable", "on",
	}
	for _, ts := range yups {
		if ts == s {
			return true
		}
	}
	return false
}
