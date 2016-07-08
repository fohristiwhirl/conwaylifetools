package main

import (
    "bufio"
    "fmt"
    "math/rand"
    "os"
    "sync"
    "time"
)

const SIZE = 64
const PATTERN_WIDTH = 8
const PATTERN_HEIGHT = 13
const CELL_CHANCE = 0.3
const ITERATIONS = 50
const THREADS = 3
const MIN_LIVING_CELLS = 16

const MIRROR = true

type Universe struct {
    cells   [64][64]int
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
        go search(n, QueryChan[n])
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


func search(thread int, my_querychan chan bool) {

    var attempt int = 0
    var work, initial Universe

    for {

        attempt += 1

        if MIRROR {
            work.setup_mirror_x()
        } else {
            work.setup_random()
        }
        work.iterate()          // Iterate at least once to clear out the junk
        work.iterate()          // But a few times more in case we got some useful precursor
        work.iterate()

        if work.count <= MIN_LIVING_CELLS {
            continue
        }

        initial = work

        for n := 0; n < ITERATIONS; n++ {

            err := work.iterate()

            if err != nil {
                // fmt.Printf("(pattern reached wall)\n")
                break
            }

            if work.count < MIN_LIVING_CELLS {
                break
            }

            if work.count == initial.count {                                        // Check cell count
                if work.left - initial.left == work.right - initial.right {         // Check width
                    if work.top - initial.top == work.bottom - initial.bottom {     // Check height
                        if work.top != initial.top || work.left != initial.left {   // Check for movement

                            if compare(&work, &initial) {
                                if MIRROR && n == 3 && (work.bottom - initial.bottom == -2 || work.bottom - initial.bottom == 2) {
                                    Console_MUTEX.Lock()
                                    fmt.Printf("(found common spaceship)\n")
                                    Console_MUTEX.Unlock()
                                    break
                                } else {
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
        }

        select {

        case <- my_querychan:
            Console_MUTEX.Lock()
            initial.dump()
            work.dump()
            Console_MUTEX.Unlock()
        default:

        }
    }
}


func (self *Universe) iterate() error {

    var newcells [SIZE][SIZE]int
    var newleft, newright, newtop, newbottom = SIZE, -1, SIZE, -1

    if self.left < 2 || self.right > SIZE - 3 || self.top < 2 || self.bottom > SIZE - 3 {
        return fmt.Errorf("iterate: pattern was at array border")
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

    return nil
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

    for x := 0 ; x < SIZE ; x++ {
        for y := 0 ; y < SIZE ; y++ {
            self.cells[x][y] = 0
        }
    }

    self.count = 0
}


func (self *Universe) setup_random() {

    self.clear_cells()

    self.left = SIZE / 2 - PATTERN_WIDTH / 2
    self.right = self.left + PATTERN_WIDTH - 1
    self.top = SIZE / 2 - PATTERN_HEIGHT / 2
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

    self.left = SIZE / 2 - PATTERN_WIDTH / 2
    self.right = self.left + PATTERN_WIDTH - 1
    self.top = SIZE / 2 - PATTERN_HEIGHT / 2
    self.bottom = self.top + PATTERN_HEIGHT - 1

    for x := self.left ; x < (SIZE) / 2 + 1; x++ {
        for y := self.top ; y <= self.bottom ; y++ {
            if rand.Float32() < CELL_CHANCE {
                self.cells[x][y] = 1
                self.cells[self.right - (x - self.left)][y] = 1
                self.count += 2
            }
        }
    }
}


func (self *Universe) setup_copperhead() {

    // This function creates a c/10 orthogonal spaceship, it can be used
    // to test that the detection function is really working.

    self.clear_cells()

    self.count = 28

    // There's gotta be a nicer way to do this...

    self.cells[22][21] = 1
    self.cells[23][21] = 1
    self.cells[26][21] = 1
    self.cells[27][21] = 1
    self.cells[24][22] = 1
    self.cells[25][22] = 1
    self.cells[24][23] = 1
    self.cells[25][23] = 1
    self.cells[21][24] = 1
    self.cells[23][24] = 1
    self.cells[26][24] = 1
    self.cells[28][24] = 1
    self.cells[21][25] = 1
    self.cells[28][25] = 1
    self.cells[21][27] = 1
    self.cells[28][27] = 1
    self.cells[22][28] = 1
    self.cells[23][28] = 1
    self.cells[26][28] = 1
    self.cells[27][28] = 1
    self.cells[23][29] = 1
    self.cells[24][29] = 1
    self.cells[25][29] = 1
    self.cells[26][29] = 1
    self.cells[24][31] = 1
    self.cells[25][31] = 1
    self.cells[24][32] = 1
    self.cells[25][32] = 1

    self.left = 21
    self.right = 28
    self.top = 21
    self.bottom = 32
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
