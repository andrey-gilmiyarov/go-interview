package main

import "fmt"

func badAppend(s []int) {
    fmt.Printf("  callee before: len=%d cap=%d\n", len(s), cap(s))
    // with s1 - no reallocation (enough cap)
    // with s2 - triggers reallocation (not enough cap)
    s = append(s, 2, 3) 
    fmt.Printf("  callee after:  len=%d cap=%d firstElemAddr=%p slice=%v\n",
        len(s), cap(s), &s[0], s)
    // NOTE: we do NOT return s
}

func main() {
    fmt.Println("\nExample 1")

    // len=1, cap=4 ➜ enough room for 2 more without realloc
    s1 := make([]int, 1, 4)
    s1[0] = 1

    fmt.Printf("caller before: len=%d cap=%d firstElemAddr=%p slice=%v\n",
        len(s1), cap(s1), &s1[0], s1)

    badAppend(s1)

    // Caller still has old header (len=1), so it "misses" appended items.
    fmt.Printf("caller after:  len=%d cap=%d firstElemAddr=%p slice=%v\n",
        len(s1), cap(s1), &s1[0], s1)

    // If the caller *happens* to reslice, it can reveal the appended items:
    s1 = s1[:3] // we *know* 2 were appended in this demo
    fmt.Printf("caller reslice: len=%d cap=%d slice=%v\n",
        len(s1), cap(s1), s1)

    fmt.Println("\nExample 2")

    // len=1, cap=1 ➜ append will reallocate
    s2 := make([]int, 1)
    s2[0] = 1

    fmt.Printf("caller before: len=%d cap=%d firstElemAddr=%p slice=%v\n",
        len(s2), cap(s2), &s2[0], s2)

    badAppend(s2)

    // Caller still has old (tiny) slice, pointing to the old array.
    fmt.Printf("caller after:  len=%d cap=%d firstElemAddr=%p slice=%v\n",
        len(s2), cap(s2), &s2[0], s2)

    // Reslicing can't help — capacity is still 1 on the caller side.
    s2 = s2[:cap(s2)]
    fmt.Printf("caller reslice: len=%d cap=%d slice=%v\n",
        len(s2), cap(s2), s2)
}
