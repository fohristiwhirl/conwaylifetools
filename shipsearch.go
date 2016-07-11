package main

import (
    "bufio"
    "fmt"
    "math/rand"
    "os"
    "sync"
    "time"
)

const WORLD_SIZE = 48
const PATTERN_WIDTH = 8
const PATTERN_HEIGHT = 8
const CELL_CHANCE = 0.3
const ITERATIONS = 12
const THREADS = 4
const MIN_LIVING_CELLS = 15

const MIRROR = false

type Universe struct {
    cells   [WORLD_SIZE][WORLD_SIZE]int
    left    int
    right   int
    top     int
    bottom  int
    count   int
}

var Console_MUTEX sync.Mutex

var QueryChan [THREADS]chan bool

// -------------------------------------------------------

func main() {
    rand.Seed(time.Now().UTC().UnixNano())

    for n := 0 ; n < THREADS ; n++ {
        QueryChan[n] = make(chan bool)
    }

    for n := 0 ; n < THREADS ; n++ {
        go random_search(n, QueryChan[n])
    }

    reader := bufio.NewReader(os.Stdin)

    for {
        Console_MUTEX.Lock()
        fmt.Print("Press enter for most recent search by thread 0...\n")
        Console_MUTEX.Unlock()

        reader.ReadString('\n')
        QueryChan[0] <- true
    }
}


func random_search(thread int, my_querychan chan bool) {

    var attempt int = 0
    var work, initial Universe

    for {

        attempt += 1

        if MIRROR {
            work.setup_mirror_x()
        } else {
            work.setup_random()
        }

        work.iterate()          // Iterate at least once to clear out the junk and get real values for .top, .bottom, .left, .right
        work.iterate()          // But a few times more in case we got some useful precursor
        work.iterate()

        if work.count < MIN_LIVING_CELLS {
            continue
        }

        initial = work

        for n := 0; n < ITERATIONS; n++ {

            work.iterate()

            if work.count < MIN_LIVING_CELLS {
                break
            }

            if work.count == initial.count {                                        // Check cell count
                if work.left - initial.left == work.right - initial.right {         // Check width
                    if work.top - initial.top == work.bottom - initial.bottom {     // Check height
                        if work.top != initial.top || work.left != initial.left {   // Check for movement
                            if compare(&work, &initial) {
                                Console_MUTEX.Lock()
                                fmt.Printf("Thread %d, #%d... Period: %d, x: %d, y: %d\n", thread, attempt, n + 1, work.left - initial.left, work.top - initial.top)
                                initial.dump()
                                Console_MUTEX.Unlock()
                                break
                            }
                        }
                    }
                }
            }
        }

        select {

        case <- my_querychan:
            Console_MUTEX.Lock()
            fmt.Printf("(failed) attempt #%d\n", attempt)
            double_dump(&initial, &work)
            Console_MUTEX.Unlock()
        default:

        }
    }
}


func (self *Universe) iterate() {

    // This doesn't change the cells at the world border

    var newcells [WORLD_SIZE][WORLD_SIZE]int
    var newleft, newright, newtop, newbottom = WORLD_SIZE, -1, WORLD_SIZE, -1

    // Because the actual algorithm goes 1 wider than the internal boundary, and then considers a further 1 cell,
    // the following is both necessary and acceptable.

    if self.left < 2 {
        self.left = 2
    }
    if self.right > WORLD_SIZE - 3 {
        self.right = WORLD_SIZE - 3
    }
    if self.top < 2 {
        self.top = 2
    }
    if self.bottom > WORLD_SIZE - 3 {
        self.bottom = WORLD_SIZE - 3
    }

    for x := self.left - 1 ; x <= self.right + 1 ; x++ {

        for y := self.top - 1 ; y <= self.bottom + 1 ; y++ {

            count :=    self.cells[x - 1][y - 1] +
                        self.cells[x - 1][y    ] +
                        self.cells[x - 1][y + 1] +

                        self.cells[x    ][y - 1] +
                        self.cells[x    ][y + 1] +

                        self.cells[x + 1][y - 1] +
                        self.cells[x + 1][y    ] +
                        self.cells[x + 1][y + 1]

            if self.cells[x][y] != 0 {          // Cell already was alive
                if count == 2 || count == 3 {
                    newcells[x][y] = 1              // Survival
                } else {
                    self.count -= 1                 // Death (no need to set the cell in the new array since it will be 0 anyway)
                }
            } else {                            // Cell was not alive
                if count == 3 {
                    newcells[x][y] = 1              // Birth
                    self.count += 1
                }
            }
            if newcells[x][y] != 0 {
                if x < newleft {
                    newleft = x
                }
                if x > newright {
                    newright = x
                }
                if y < newtop {
                    newtop = y
                }
                if y > newbottom {
                    newbottom = y
                }
            }
        }
    }

    self.cells = newcells

    self.left = newleft
    self.right = newright
    self.top = newtop
    self.bottom = newbottom
}


func (self *Universe) dump() {

    var s string

    for y := self.top ; y <= self.bottom ; y++ {
        for x := self.left ; x <= self.right ; x++ {
            if self.cells[x][y] != 0 {
                s = "O"
            } else {
                s = "."
            }
            fmt.Printf("%s", s)
        }
        fmt.Printf("\n")
    }
    fmt.Printf("\n")
}


func (self *Universe) clear_cells() {       // Note: doesn't fix left/right/top/bottom vars

    for x := 0 ; x < WORLD_SIZE ; x++ {
        for y := 0 ; y < WORLD_SIZE ; y++ {
            self.cells[x][y] = 0
        }
    }

    self.count = 0
}


func (self *Universe) setup_random() {

    self.clear_cells()

    self.left = WORLD_SIZE / 2 - PATTERN_WIDTH / 2
    self.right = self.left + PATTERN_WIDTH - 1
    self.top = WORLD_SIZE / 2 - PATTERN_HEIGHT / 2
    self.bottom = self.top + PATTERN_HEIGHT - 1

    for x := self.left ; x <= self.right ; x++ {
        for y := self.top ; y <= self.bottom ; y++ {
            if rand.Float32() < CELL_CHANCE {
                self.cells[x][y] = 1
                self.count += 1
            }
        }
    }
}


func (self *Universe) setup_mirror_x() {

    self.clear_cells()

    self.left = WORLD_SIZE / 2 - PATTERN_WIDTH / 2
    self.right = self.left + PATTERN_WIDTH - 1
    self.top = WORLD_SIZE / 2 - PATTERN_HEIGHT / 2
    self.bottom = self.top + PATTERN_HEIGHT - 1

    for x := self.left ; x < (WORLD_SIZE) / 2 + 1; x++ {
        for y := self.top ; y <= self.bottom ; y++ {
            if rand.Float32() < CELL_CHANCE {
                self.cells[x][y] = 1
                self.count += 1

                other_x := self.right - (x - self.left)
                if other_x != x {                           // Can be false if odd pattern width and we are on midline
                    self.cells[other_x][y] = 1
                    self.count += 1
                }
            }
        }
    }
}


func compare(one *Universe, two *Universe) bool {       // Assumes patterns have same-sized boundary box inside
    dx := two.left - one.left
    dy := two.top - one.top
    for x := one.left ; x <= one.right ; x++ {
        for y := one.top ; y <= one.bottom ; y++ {
            if one.cells[x][y] != two.cells[x + dx][y + dy] {
                return false
            }
        }
    }
    return true
}


func double_dump(one *Universe, two *Universe) {

    var s string

    fmt.Printf("%d %d %d %d - %d %d %d %d\n", one.left, one.top, one.right, one.bottom, two.left, two.top, two.right, two.bottom)

    for y := 0 ; y < WORLD_SIZE ; y++ {
        for x := 0 ; x < WORLD_SIZE ; x++ {
            if one.cells[x][y] != 0 {
                s = "O"
            } else {
                s = "."
            }
            fmt.Printf("%s", s)
        }

        fmt.Printf("   ")

        for x := 0 ; x < WORLD_SIZE ; x++ {
            if two.cells[x][y] != 0 {
                s = "O"
            } else {
                s = "."
            }
            fmt.Printf("%s", s)
        }

        fmt.Printf("\n")
    }

    fmt.Printf("Counts: %d, %d\n", one.count, two.count)
}
