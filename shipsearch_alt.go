// An alternative shipsearcher, but seems slower for whatever reason

package main

import (
    "bufio"
    "fmt"
    "math/rand"
    "os"
    "sync"
    "time"
)

const HEIGHT = 64

const PATTERN_WIDTH = 6
const PATTERN_HEIGHT = 6

const ITERATIONS = 12
const PREITERATIONS = 2
const MINIMUM_PERIOD = 6
const MAX_X_TRAVEL = 2
const MIN_CELLS = 10            // A hwss has 18 at most, a mwss has 15

const THREADS = 4


var QueryChan = make(chan bool)
var Console_MUTEX sync.Mutex


type World struct {
    top int
    lines [HEIGHT]uint64
    bottom int
}


type BareWorld struct {
    lines [HEIGHT]uint64
}


func hasBit(n uint64, pos uint8) bool {     // Will the compiler inline this? If not, do it ourselves later
    val := n & (1 << pos)
    return (val > 0)
}


func print_line(line uint64) {
    var x uint8

    for x = 0 ; x < 64 ; x++ {
        if hasBit(line, x) {
            fmt.Printf("O")
        } else {
            fmt.Printf(".")
        }
    }
    fmt.Printf("\n")
}


func iterate_line(above uint64, line uint64, below uint64) uint64 {

    // Ultimately this function should use some large lookup table that
    // perhaps looks up a 8x3 region to make the middle 6 cells, or such

    var x uint8
    var n uint8
    var neighbours uint8
    var result uint64

    for x = 0 ; x < 64 ; x++ {

        neighbours = 0

        if x == 0 {                                             // Left edge case

            for n = 0 ; n <= 1 ; n++ {
                if hasBit(above, n) {
                    neighbours += 1
                }
                if hasBit(below, n) {
                    neighbours += 1
                }
            }
            if hasBit(line, 1) {
                neighbours += 1
            }

        } else if x == 63 {                                     // Right edge case

            for n = 62 ; n <= 63 ; n++ {
                if hasBit(above, n) {
                    neighbours += 1
                }
                if hasBit(below, n) {
                    neighbours += 1
                }
            }
            if hasBit(line, 62) {
                neighbours += 1
            }

        } else {                                                // General case

            for n = x - 1 ; n <= x + 1 ; n++ {
                if hasBit(above, n) {
                    neighbours += 1
                }
                if hasBit(below, n) {
                    neighbours += 1
                }
            }
            if hasBit(line, x - 1) {
                neighbours += 1
            }
            if hasBit(line, x + 1) {
                neighbours += 1
            }

        }

        if hasBit(line, x) {
            if neighbours == 2 || neighbours == 3 {
                result += 1 << x
            }
        } else {
            if neighbours == 3 {
                result += 1 << x
            }
        }
    }

    return result
}


func (world *World) print() {
    for y := world.top ; y <= world.bottom ; y++ {
        print_line(world.lines[y])
    }
}


func (world *World) count() int {
    result := 0

    for x := uint8(0) ; x < 64 ; x++ {
        for y := 0 ; y < 64 ; y++ {
            if hasBit(world.lines[y], x) {
                result += 1
            }
        }
    }

    return result
}


func (world *World) iterate() {
    var line_before_iteration uint64
    var old_line_above uint64

    var start_y int
    var end_y int

    var y int

    var new_top int = HEIGHT
    var new_bottom int = -1

    if world.top > 0 {
        start_y = world.top - 1
    } else {
        start_y = 0
    }

    if world.bottom <= HEIGHT - 2 {
        end_y = world.bottom + 1
    } else {
        end_y = HEIGHT - 1
    }

    // Iteration algo proper...

    old_line_above = 0

    for y = start_y ; y <= end_y - 1 ; y++ {       // "y <= end_y - 1" is correct; the actual line of end_y is done as a special case

        line_before_iteration = world.lines[y]
        world.lines[y] = iterate_line(old_line_above, world.lines[y], world.lines[y + 1])
        old_line_above = line_before_iteration

        if y < new_top && world.lines[y] > 0 {
            new_top = y
        }
        if y > new_bottom && world.lines[y] > 0 {
            new_bottom = y
        }
    }

    // Special case of last line...

    world.lines[y] = iterate_line(old_line_above, world.lines[y], 0)

    if y < new_top && world.lines[y] > 0 {
        new_top = y
    }
    if y > new_bottom && world.lines[y] > 0 {
        new_bottom = y
    }

    world.top = new_top
    world.bottom = new_bottom
}


func (w *BareWorld) print() {

    var world World
    world.lines = w.lines

    for y := 0 ; y < HEIGHT ; y++ {
        if world.lines[y] > 0 {
            world.top = y
            break
        }
    }

    for y := HEIGHT - 1 ; y >= 0 ; y-- {
        if world.lines[y] > 0 {
            world.bottom = y
            break
        }
    }

    for y := world.top ; y <= world.bottom ; y++ {
        print_line(world.lines[y])
    }
}


func run(w *BareWorld) {    // Create a copy of the world, run it, and compare. Report if success.
                            // The copy is a full" world with top and bottom variables.
    var world World
    world.lines = w.lines

    // Set the world top and bottom vars...

    for y := 0 ; y < HEIGHT ; y++ {
        if world.lines[y] > 0 {
            world.top = y
            break
        }
    }

    for y := HEIGHT - 1 ; y >= 0 ; y-- {
        if world.lines[y] > 0 {
            world.bottom = y
            break
        }
    }

    // Run preiterations...

    for n := 0 ; n < PREITERATIONS ; n++ {
        world.iterate()
    }

    var initialworld World
    initialworld = world

    // Actual run...

    run:
    for n := 0 ; n < ITERATIONS ; n++ {
        world.iterate()

        if n >= MINIMUM_PERIOD - 1 {

            if initialworld.top - world.top == initialworld.bottom - world.bottom {

                y_offset := initialworld.top - world.top

                for x_offset := -MAX_X_TRAVEL ; x_offset <= MAX_X_TRAVEL ; x_offset++ {

                    for y := world.top ; y <= world.bottom ; y++ {

                        var newline uint64
                        var oldline uint64

                        if x_offset < 0 {
                            newline = world.lines[y] << uint(-x_offset)
                        } else {
                            newline = world.lines[y] >> uint(x_offset)
                        }
                        oldline = initialworld.lines[y + y_offset]
                        if newline != oldline {
                            break
                        }

                        if y == world.bottom {                      // We checked all lines and they matched
                            if x_offset != 0 || y_offset != 0 {
                                if world.count() >= MIN_CELLS {
                                    Console_MUTEX.Lock()
                                    world.print()
                                    fmt.Printf("RESULT! Period: %d\n", n + 1)
                                    Console_MUTEX.Unlock()
                                }
                                break run
                            }
                        }
                    }
                }
            }
        }
    }
}


func depth_first_search(world *BareWorld, y int, endy int) {

    var upper uint64
    var val uint64

    upper = 1 << PATTERN_WIDTH      // in Python this would be:     upper = 2 ** PATTERN_WIDTH

    for val = 0 ; val < upper ; val++ {

        world.lines[y] = val
        world.lines[y] <<= (64 - PATTERN_WIDTH) / 2

        if y == endy {
            run(world)

            select {
            case <- QueryChan:
                Console_MUTEX.Lock()
                world.print()
                Console_MUTEX.Unlock()
            default:
            }
        }

        if y < endy {
            depth_first_search(world, y + 1, endy)
        }
    }
}


func random_search(world *BareWorld) {

    var attempts uint64 = 0

    var start_y int = HEIGHT / 2 - PATTERN_HEIGHT / 2
    var end_y int = start_y + PATTERN_HEIGHT - 1
    var upper int64 = 1 << PATTERN_WIDTH

    for {

        attempts += 1

        for y := start_y ; y <= end_y ; y++ {

            world.lines[y] = uint64(rand.Int63n(upper))
            world.lines[y] &= uint64(rand.Int63n(upper))

            // After the above lines, 1 in 4 cells should be on

            world.lines[y] <<= (64 - PATTERN_WIDTH) / 2
        }

        run(world)

        select {
        case <- QueryChan:
            Console_MUTEX.Lock()
            fmt.Printf("Attempt %d in this thread:\n", attempts)
            world.print()
            Console_MUTEX.Unlock()
        default:
        }
    }
}


func search() {

    var world BareWorld

    var start_y int = HEIGHT / 2 - PATTERN_HEIGHT / 2
    var end_y int = start_y + PATTERN_HEIGHT - 1

    depth_first_search(&world, start_y, end_y)
    // random_search(&world)
}


func main() {

    for n := 0 ; n < THREADS ; n++ {
        go search()
    }

    reader := bufio.NewReader(os.Stdin)

    for {
        Console_MUTEX.Lock()
        fmt.Print("\nPress enter for most recent search...\n")
        Console_MUTEX.Unlock()

        reader.ReadString('\n')
        QueryChan <- true
    }
}


func init() {
    rand.Seed(time.Now().UTC().UnixNano())
}
