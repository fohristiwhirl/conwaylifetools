# Convert:
#    http://www.conwaylife.com/wiki/Run_Length_Encoded
# To:
#    http://www.conwaylife.com/wiki/Apgsearch_format

import sys


encoding = "0123456789abcdefghijklmnopqrstuv"


def parse_rle(infile):

    # Note that, when parsing numbers, they can be more than one character long

    worldlines = [[]]

    maxlen = 0

    number = 0      # How many cells to add. If 0 then 1 is used.

    for line in infile:
        if line.startswith("#"):
            continue

        if line.startswith("x"):     # if re.search(r'x = \d+, y = \d+', line):
            continue

        for c in line:
            if c.isspace():
                continue

            if c == "!":
                break

            if c.isdigit():
                number = number * 10 + int(c)

            if c in "bo":
                if number == 0:
                    number = 1
                for n in range(number):
                    worldlines[-1].append(c)
                number = 0

            if c == "$":
                if len(worldlines[-1]) > maxlen:
                    maxlen = len(worldlines[-1])
                worldlines.append([])

    for line in worldlines:
        missing = maxlen - len(line)
        for n in range(missing):
            line.append("b")

    return worldlines


def print_world(world):
    for line in world:
        for c in line:
            if c == "b":
                print(".", end="")
            else:
                print("0", end="")
        print()
    print()


def make_world_y_mod_5(world):

    # Pads a world so that the number of lines is divisible by 5

    missing = (5 - len(world) % 5) % 5

    for n in range(missing):
        world.append([])
        for i in range(len(world[0])):
            world[-1].append("b")


def world_to_apg(world):

    make_world_y_mod_5(world)

    width = len(world[0])

    s = ""
    li = 0
    while 1:
        if len(world) <= li:
            return s

        if s:
            s += "z"

        for x in range(width):
            binary = 0
            if world[li][x] == "o":
                binary += 1
            if world[li + 1][x] == "o":
                binary += 2
            if world[li + 2][x] == "o":
                binary += 4
            if world[li + 3][x] == "o":
                binary += 8
            if world[li + 4][x] == "o":
                binary += 16

            s += encoding[binary]

        li += 5

    return s


def main():
    f = open(sys.argv[1])
    world = parse_rle(f)
    print_world(world)
    result = world_to_apg(world)
    print(result)
    input()


if __name__ == "__main__":
    main()
