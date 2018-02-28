package geo

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
)

// Represents a Physical Point in geographic notation [lat, lng].
type Point struct {
	lat float64
	lng float64
}

const (
	// According to Wikipedia, the Earth's radius is about 6,371km
	EARTH_RADIUS = 6371
)

type Format int

const (
	// Decimal degrees format, e.g. 45.699750,-69.733722
	DecimalDegrees = iota
	// Decimal minutes format, e.g. N 45 41.985, W 69 44.023
	DecimalMinutes
	// Decimal seconds format, e.g. N 45 41 59.100, W 69 41 1.399
	DecimalSeconds
)

var formatRegex *regexp.Regexp

func init() {
	// formatRegex matches three different text formats for latitude/longitude vales
	formatRegex =
		regexp.MustCompile(
			// Decimal degrees, e.g. 45.699958,-69.733729 or N 45.699958 W 69.733729
			// or 45.699958 N 69.733729 W
			`^\s*(?P<ns>[NS+-]?)\s*(?P<lat_deg>\d{1,2}(?:\.\d*)?)°?\s*(?P<ns2>[NS]?)` +
				`(?:\s+|\s*,\s*)` +
				`(?P<ew>[EW+-]?)\s*(?P<lon_deg>\d{1,3}(?:\.\d*)?)°?\s*(?P<ew2>[EW]?)\s*$|` +
				// Decimal minutes, e.g. 45 41.997, -69 44.024 or N 45 41.997 W 69 44.024
				`^\s*(?P<ns>[NS+-]?)\s*(?P<lat_deg>\d{1,2})°?\s+(?P<lat_min>\d{1,2}(?:\.\d*)?)'?\s*(?P<ns2>[NS]?)` +
				`(?:\s+|\s*,\s*)` +
				`(?P<ew>[EW+-]?)\s*(?P<lon_deg>\d{1,3})°?\s+(?P<lon_min>\d{1,2}(?:\.\d*)?)'?\s*(?P<ew2>[EW]?)\s*$|` +
				// Decimal seconds, e.g. 45 41 59.85, -69 44 01.42 or N 45 41 59.85, W 69 44 01.42
				`^\s*(?P<ns>[NS+-]?)\s*(?P<lat_deg>\d{1,2})°?\s+(?P<lat_min>\d{1,2})'?\s+(?P<lat_sec>\d{1,2}(?:\.\d*)?)"?\s*(?P<ns2>[NS]?)` +
				`(?:\s+|\s*,\s*)` +
				`(?P<ew>[EW+-]?)\s*(?P<lon_deg>\d{1,3})°?\s+(?P<lon_min>\d{1,2})'?\s+(?P<lon_sec>\d{1,2}(?:\.\d*)?)"?\s*(?P<ew2>[EW]?)\s*$`)
}

// Returns a new Point populated by the passed in latitude (lat) and longitude (lng) values.
func NewPoint(lat float64, lng float64) *Point {
	return &Point{lat: lat, lng: lng}
}

// Parses a longitude/latitude string in a variety of formats and
// returns a new Point populated with the parsed values.
func Parse(value string) (*Point, error) {
	segments := formatRegex.FindStringSubmatch(value)
	if len(segments) < 1 {
		return nil, errors.New("Unable to parse value: " + value)
	}
	switch {
	case segments[2] != "":
		lat, err := calcValue(segments[1:4])
		if err != nil {
			return nil, err
		}
		lng, err := calcValue(segments[4:7])
		if err != nil {
			return nil, err
		}
		return NewPoint(lat, lng), err
	case segments[8] != "":
		lat, err := calcValue(segments[7:11])
		if err != nil {
			return nil, err
		}
		lng, err := calcValue(segments[11:15])
		if err != nil {
			return nil, err
		}
		return NewPoint(lat, lng), err
	case segments[16] != "":
		lat, err := calcValue(segments[15:20])
		if err != nil {
			return nil, err
		}
		lng, err := calcValue(segments[20:25])
		if err != nil {
			return nil, err
		}
		return NewPoint(lat, lng), err
	default:
		return nil, errors.New("Unable to parse value: " + value)
	}
	return nil, nil
}
func trim(segments []string) []string {
	start := -1
	end := len(segments)
	for i, s := range segments {
		if s != "" && start == -1 {
			start = i
		}
		if s != "" && start > -1 {
			end = i
		}
	}
	return segments[start : end+1]
}
func calcValue(segments []string) (value float64, err error) {
	sign := 1.0
	last := len(segments) - 1
	if segments[0] == "S" || segments[0] == "W" || segments[0] == "-" ||
		segments[last] == "S" || segments[last] == "W" {
		sign = -1.0
	}
	value = 0.0
	divisor := 1.0
	for _, s := range segments[1:last] {
		sv, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return value, err
		}
		value = value + sv/divisor
		divisor = divisor * 60.0
	}
	return value * sign, err
}

// Formats a Point into one of several common string formats
func (p *Point) Format(format Format) (string, error) {
	switch format {
	case DecimalDegrees:
		return fmt.Sprintf("%f,%f", p.lat, p.lng), nil
	case DecimalMinutes:
		ns := "N"
		if p.lat < 0 {
			ns = "S"
		}
		lati, latf := math.Modf(math.Abs(p.lat))
		latd := int(lati)
		latm := latf * 60.0
		ew := "E"
		if p.lng < 0 {
			ew = "W"
		}
		lngi, lngr := math.Modf(math.Abs(p.lng))
		lngd := int(lngi)
		lngm := lngr * 60.0
		return fmt.Sprintf("%s %d %.3f, %s %d %.3f", ns, latd, latm, ew, lngd, lngm), nil
	case DecimalSeconds:
		ns := "N"
		if p.lat < 0 {
			ns = "S"
		}
		lati, latf := math.Modf(math.Abs(p.lat))
		latd := int(lati)
		latmf := latf * 60.0
		lati, latf = math.Modf(latmf)
		latm := int(lati)
		lats := latf * 60.0
		ew := "E"
		if p.lng < 0 {
			ew = "W"
		}
		lngi, lngf := math.Modf(math.Abs(p.lng))
		lngd := int(lngi)
		lngmf := lngf * 60.0
		lngi, lngf = math.Modf(lngmf)
		lngm := int(lati)
		lngs := lngf * 60.0
		return fmt.Sprintf("%s %d %d %.3f, %s %d %d %.3f", ns, latd, latm, lats, ew, lngd, lngm, lngs), nil
	default:
		return "", errors.New("Invalid format: " + string(format))
	}
}

// Returns Point p's latitude.
func (p *Point) Lat() float64 {
	return p.lat
}

// Returns Point p's longitude.
func (p *Point) Lng() float64 {
	return p.lng
}

// Returns a Point populated with the lat and lng coordinates
// by transposing the origin point the passed in distance (in kilometers)
// by the passed in compass bearing (in degrees).
// Original Implementation from: http://www.movable-type.co.uk/scripts/latlong.html
func (p *Point) PointAtDistanceAndBearing(dist float64, bearing float64) *Point {

	dr := dist / EARTH_RADIUS

	bearing = (bearing * (math.Pi / 180.0))

	lat1 := (p.lat * (math.Pi / 180.0))
	lng1 := (p.lng * (math.Pi / 180.0))

	lat2_part1 := math.Sin(lat1) * math.Cos(dr)
	lat2_part2 := math.Cos(lat1) * math.Sin(dr) * math.Cos(bearing)

	lat2 := math.Asin(lat2_part1 + lat2_part2)

	lng2_part1 := math.Sin(bearing) * math.Sin(dr) * math.Cos(lat1)
	lng2_part2 := math.Cos(dr) - (math.Sin(lat1) * math.Sin(lat2))

	lng2 := lng1 + math.Atan2(lng2_part1, lng2_part2)
	lng2 = math.Mod((lng2+3*math.Pi), (2*math.Pi)) - math.Pi

	lat2 = lat2 * (180.0 / math.Pi)
	lng2 = lng2 * (180.0 / math.Pi)

	return &Point{lat: lat2, lng: lng2}
}

// Calculates the Haversine distance between two points in kilometers.
// Original Implementation from: http://www.movable-type.co.uk/scripts/latlong.html
func (p *Point) GreatCircleDistance(p2 *Point) float64 {
	dLat := (p2.lat - p.lat) * (math.Pi / 180.0)
	dLon := (p2.lng - p.lng) * (math.Pi / 180.0)

	lat1 := p.lat * (math.Pi / 180.0)
	lat2 := p2.lat * (math.Pi / 180.0)

	a1 := math.Sin(dLat/2) * math.Sin(dLat/2)
	a2 := math.Sin(dLon/2) * math.Sin(dLon/2) * math.Cos(lat1) * math.Cos(lat2)

	a := a1 + a2

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return EARTH_RADIUS * c
}

// Calculates the initial bearing (sometimes referred to as forward azimuth)
// Original Implementation from: http://www.movable-type.co.uk/scripts/latlong.html
func (p *Point) BearingTo(p2 *Point) float64 {

	dLon := (p2.lng - p.lng) * math.Pi / 180.0

	lat1 := p.lat * math.Pi / 180.0
	lat2 := p2.lat * math.Pi / 180.0

	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) -
		math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)
	brng := math.Atan2(y, x) * 180.0 / math.Pi

	if brng < 0. {
		brng = 360. + brng
	}

	return brng
}

// Calculates the midpoint between 'this' point and the supplied point.
// Original implementation from http://www.movable-type.co.uk/scripts/latlong.html
func (p *Point) MidpointTo(p2 *Point) *Point {
	lat1 := p.lat * math.Pi / 180.0
	lat2 := p2.lat * math.Pi / 180.0

	lon1 := p.lng * math.Pi / 180.0
	dLon := (p2.lng - p.lng) * math.Pi / 180.0

	bx := math.Cos(lat2) * math.Cos(dLon)
	by := math.Cos(lat2) * math.Sin(dLon)

	lat3Rad := math.Atan2(
		math.Sin(lat1)+math.Sin(lat2),
		math.Sqrt(math.Pow(math.Cos(lat1)+bx, 2)+math.Pow(by, 2)),
	)
	lon3Rad := lon1 + math.Atan2(by, math.Cos(lat1)+bx)

	lat3 := lat3Rad * 180.0 / math.Pi
	lon3 := lon3Rad * 180.0 / math.Pi

	return NewPoint(lat3, lon3)
}

// Renders the current point to a byte slice.
// Implements the encoding.BinaryMarshaler Interface.
func (p *Point) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	err := binary.Write(&buf, binary.LittleEndian, p.lat)
	if err != nil {
		return nil, fmt.Errorf("unable to encode lat %v: %v", p.lat, err)
	}
	err = binary.Write(&buf, binary.LittleEndian, p.lng)
	if err != nil {
		return nil, fmt.Errorf("unable to encode lng %v: %v", p.lng, err)
	}

	return buf.Bytes(), nil
}

func (p *Point) UnmarshalBinary(data []byte) error {
	buf := bytes.NewReader(data)

	var lat float64
	err := binary.Read(buf, binary.LittleEndian, &lat)
	if err != nil {
		return fmt.Errorf("binary.Read failed: %v", err)
	}

	var lng float64
	err = binary.Read(buf, binary.LittleEndian, &lng)
	if err != nil {
		return fmt.Errorf("binary.Read failed: %v", err)
	}

	p.lat = lat
	p.lng = lng
	return nil
}

// Renders the current Point to valid JSON.
// Implements the json.Marshaller Interface.
func (p *Point) MarshalJSON() ([]byte, error) {
	res := fmt.Sprintf(`{"lat":%v, "lng":%v}`, p.lat, p.lng)
	return []byte(res), nil
}

// Decodes the current Point from a JSON body.
// Throws an error if the body of the point cannot be interpreted by the JSON body
func (p *Point) UnmarshalJSON(data []byte) error {
	// TODO throw an error if there is an issue parsing the body.
	dec := json.NewDecoder(bytes.NewReader(data))
	var values map[string]float64
	err := dec.Decode(&values)

	if err != nil {
		log.Print(err)
		return err
	}

	*p = *NewPoint(values["lat"], values["lng"])

	return nil
}
