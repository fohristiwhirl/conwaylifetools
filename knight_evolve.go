// A spectacularly unsuccessful attempt to evolve a knightship using
// parallel threads with metropolis-coupling or something like it

package main

import (
    "bufio"
    "fmt"
    "math/rand"
    "os"
    "sync"
    "time"
)

const PATTERN_WIDTH = 24
const PATTERN_HEIGHT = 24
const WORLD_SIZE = 36
const INITIAL_CELL_CHANCE = 0.15
const ITERATIONS = 6
const THREADS = 6

const MIN_CELLS = 80
const MAX_CELLS = 100

type Universe struct {
    cells   [WORLD_SIZE][WORLD_SIZE]int
    left    int
    right   int
    top     int
    bottom  int
    count   int
}

type Report struct {
    score   int
    ptr     *Universe
}

var Console_MUTEX sync.Mutex

var QueryChan [THREADS]chan bool
var ReportChan [THREADS]chan Report
var PtrChan [THREADS]chan *Universe


var Heat =  [...]float32{
                    0,
                    0.00001,
                    0.0001,
                    0.0005,
                    0.001,
                    0.01,
                    }

// -------------------------------------------------------

func main() {
    rand.Seed(time.Now().UTC().UnixNano())

    for n := 0 ; n < THREADS ; n++ {
        QueryChan[n] = make(chan bool)
        ReportChan[n] = make(chan Report)
        PtrChan[n] = make(chan *Universe)
    }

    for n := 0 ; n < THREADS ; n++ {
        go evolve(n)
    }

    go hub()

    reader := bufio.NewReader(os.Stdin)

    for {
        Console_MUTEX.Lock()
        fmt.Print("Press enter to query thread 0...\n")
        Console_MUTEX.Unlock()

        reader.ReadString('\n')
        QueryChan[0] <- true
    }
}


func hub() {

    var iteration int = 0

    var reports [THREADS]Report

    for {

        iteration += 1

        for n := 0 ; n < THREADS ; n++ {
            reports[n] = <- ReportChan[n]
        }

        for n := 0 ; n < THREADS - 1 ; n++ {
            if reports[n].score >= reports[n + 1].score {
                tmp := reports[n]
                reports[n] = reports[n + 1]
                reports[n + 1] = tmp
            }
        }

        for n := 0 ; n < THREADS ; n++ {
            PtrChan[n] <- reports[n].ptr
        }

        if iteration % 5000 == 0 {
            Console_MUTEX.Lock()
            for n:= 0 ; n < THREADS ; n++ {
                fmt.Printf("%3d ", reports[n].score)
            }
            fmt.Printf("\n")
            Console_MUTEX.Unlock()
        }
    }
}


func evolve(thread int) {

    var attempt int = 0
    var mutations = 0
    var swaps = 0
    var initial, work Universe
    var stable *Universe
    var score = 999999
    var report Report

    stable = new(Universe)

    stable.setup_random()

    for {
        attempt += 1

        initial = *stable
        initial.mutate()
        work = initial

        for n := 0 ; n < ITERATIONS ; n++ {
            work.iterate()
        }

        newscore := fitness(&initial, &work)
        if newscore < score || (rand.Float32() < Heat[thread]) {
            score = newscore
            *stable = initial
            mutations += 1
        }

        report.score = score
        report.ptr = stable

        ReportChan[thread] <- report

        newstable := <- PtrChan[thread]
        if newstable != stable {
            swaps += 1
            stable = newstable
        }

        // FIXME: Print some sort of status when asked...

        select {
        case <- QueryChan[thread]:
            Console_MUTEX.Lock()
            double_dump(&initial, &work)
            fmt.Printf("Attempt: %d Score: %d Initial: %d Final: %d Mutations: %d Swaps: %d Ptr: %p\n",
                attempt, score, initial.count, work.count, mutations, swaps, stable)
            Console_MUTEX.Unlock()
        default:
        }
    }
}


func (self *Universe) iterate() error {

    var newcells [WORLD_SIZE][WORLD_SIZE]int
    var newleft, newright, newtop, newbottom = WORLD_SIZE, -1, WORLD_SIZE, -1

    if self.left < 2 || self.right > WORLD_SIZE - 3 || self.top < 2 || self.bottom > WORLD_SIZE - 3 {
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

    for y := 0 ; y < WORLD_SIZE ; y++ {
        for x := 0 ; x < WORLD_SIZE ; x++ {
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
            if rand.Float32() < INITIAL_CELL_CHANCE {
                self.cells[x][y] = 1
                self.count += 1
            }
        }
    }
}


func fitness(initial *Universe, final *Universe) int {      // Lower is better

    bad_cells := initial.count + final.count
    good_cells := 0

    count_mismatch := initial.count - final.count
    if count_mismatch < 0 {
        count_mismatch *= -1
    }

    for x := initial.left ; x <= initial.right ; x++ {
        for y := initial.top ; y <= initial.bottom ; y++ {
            if initial.cells[x][y] == 1 {
                if final.cells[x + 2][y + 1] == 1 {
                    bad_cells -= 1
                    good_cells += 1
                }
            }
        }
    }

    return bad_cells + count_mismatch - good_cells
}


func compare(one *Universe, two *Universe) bool {
    dx := two.left - one.left
    dy := two.top - one.top

    if dx != two.right - one.right || dy != two.bottom - one.bottom {
        return false
    }

    for x := one.left ; x <= one.right ; x++ {
        for y := one.top ; y <= one.bottom ; y++ {
            if one.cells[x][y] != two.cells[x + dx][y + dy] {
                return false
            }
        }
    }

    return true
}


func (self *Universe) mutate() {

    var needed_births int
    var needed_deaths int

    for {

        if self.count < MIN_CELLS {
            needed_births = 1 + MIN_CELLS - self.count
            needed_deaths = 1
        } else if self.count > MAX_CELLS {
            needed_births = 1
            needed_deaths = 1 + self.count - MAX_CELLS
        } else {
            needed_births = int(rand.Int31n(4))
            needed_deaths = int(rand.Int31n(4))
        }

        for {

            if needed_births <= 0 && needed_deaths <= 0 {
                return
            }

            x := rand.Int31n(PATTERN_WIDTH) + (WORLD_SIZE - PATTERN_WIDTH) / 2
            y := rand.Int31n(PATTERN_HEIGHT) + (WORLD_SIZE - PATTERN_HEIGHT) / 2

            if self.cells[x][y] == 1 && needed_deaths > 0 {
                self.cells[x][y] = 0
                self.count -= 1
                needed_deaths -= 1
            } else if self.cells[x][y] == 0 && needed_births > 0 {
                self.cells[x][y] = 1
                self.count += 1
                needed_births -= 1
            }
        }
    }
}


/*
func (self *Universe) cleanup() {   // Can leave the universe with oversized boundary box but that's OK?

    for x := self.left ; x <= self.right ; x++ {
        for y := self.top ; y <= self.bottom ; y++ {

            if self.cells[x][y] == 1 {
                if self.cells[x - 1][y - 1] == 0 &&
                   self.cells[x - 1][y    ] == 0 &&
                   self.cells[x - 1][y + 1] == 0 &&
                   self.cells[x    ][y - 1] == 0 &&
                   self.cells[x    ][y + 1] == 0 &&
                   self.cells[x + 1][y - 1] == 0 &&
                   self.cells[x + 1][y    ] == 0 &&
                   self.cells[x + 1][y + 1] == 0 {

                    self.cells[x][y] = 0
                    self.count -= 1
                }
            }
        }
    }
}
*/


func double_dump(one *Universe, two *Universe) {

    var s string

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

    fmt.Printf("\n")
}
