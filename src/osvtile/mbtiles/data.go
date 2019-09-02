package mbtiles

import (
    "database/sql"
    "fmt"
    _ "github.com/mattn/go-sqlite3"
    "log"
    "strconv"
    "strings"
)

// Position represents a lon/lat/zoom
type Position [3]float64

func (p Position) Lon() float64 {
    return p[0]
}

func (p Position) Lat() float64 {
    return p[1]
}

func (p Position) Zoom() int {
    return int(p[2])
}

// BBox is a bounding box representation of WGS 84 values: left, bottom, right, top
type BBox [4]float64

func (b BBox) Left() float64 {
    return b[0]
}

func (b BBox) Bottom() float64 {
    return b[1]
}

func (b BBox) Right() float64 {
    return b[2]
}

func (b BBox) Top() float64 {
    return b[3]
}

// Version specifies the MBTiles information (as specified in the spec)
type Version struct {
    Name    string
    Format  string
    Bounds  BBox
    Center  Position
    Maxzoom int
    Minzoom int
    JSON    string
    Meta    map[string]string
}

func (v *Version) String() string {
    return fmt.Sprintf("version: {name = %s, format = %s}", v.Name, v.Format)
}

// MBTiles holds the datasource for tile information
type MBTiles struct {
    // the underlying mbtiles package
    db sql.DB
}

// FetchTile will query the package to return a given tile at the specified location and zoom. If no tile is found
// this func will return a `nil,nil`
func (m *MBTiles) FetchTile(x, y, z int) ([]byte, error) {
    var tile []byte

    err := m.db.QueryRow(
        "select tile_data from tiles where zoom_level = ? and tile_column = ? and tile_row = ?",
        z, x, y,
    ).Scan(&tile)

    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }

        return nil, err
    }

    return tile, nil
}

// Version will report the underlying MBTiles version information
func (m *MBTiles) Version() (*Version, error) {
    rows, err := m.db.Query("select * from metadata")
    if err != nil {
        return nil, err
    }

    defer func() {
        err := rows.Close()

        if err != nil {
            log.Printf("error closing db rows: error = %s", err)
        }
    }()

    v := &Version{
        Meta: map[string]string{},
    }

    for rows.Next() {
        var key string
        var value string
        err = rows.Scan(&key, &value)
        if err != nil {
            return nil, err
        }

        switch key {
        case "name":
            v.Name = value
        case "format":
            v.Format = value
        case "maxzoom":
            if v.Maxzoom, err = strconv.Atoi(value); err != nil {
                return nil, fmt.Errorf("failed to parse maxzoom: error = %s", err)
            }
        case "minzoom":
            if v.Minzoom, err = strconv.Atoi(value); err != nil {
                return nil, fmt.Errorf("failed to parse minzoom: error = %s", err)
            }
        case "json":
            v.JSON = value
        case "center":
            p, err := parsePosition(value)

            if err != nil {
                return nil, err
            }

            v.Center = *p
        default:
            v.Meta[key] = value
        }
    }

    err = rows.Err()
    if err != nil {
        return nil, err
    }

    return v, nil
}

// Close will shutdown the MBTiles tile source
func (m *MBTiles) Close() error {
    return m.db.Close()
}

// NewMVT will construct a new tile source dataset
func NewMVT(path string) (*MBTiles, error) {
    db, err := sql.Open("sqlite3", fmt.Sprintf("%s?mode=ro&_query_only=true&_mutex=no", path))
    if err != nil {
        return nil, err
    }

    if err = db.Ping(); err != nil {
        return nil, err
    }

    log.Printf("created new MBTiles tile source: path = %s", path)
    return &MBTiles{
        db: *db,
    }, nil
}

// parses a value such as `-0.173,51.3859,10` to a position type
func parsePosition(value string) (*Position, error) {
    parts := strings.Split(value, ",")

    p := Position{}

    for i, part := range parts {
        f, err := strconv.ParseFloat(strings.TrimSpace(part), 64)

        if err != nil {
            return nil, err
        }

        p[i] = f
    }

    return &p, nil
}
