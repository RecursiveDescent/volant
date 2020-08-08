# Arrays

Arrays are linear memory structures representing a group of values. Arrays in `Volant` have type `type[num]`.
For example, `array: u8[10]` declares an array of 10 `u8`s.

Arrays are stack based, so 10 bytes are pushed direclty to stack when the `array` is declared but the values are not initialized.

Individual elements of an array are used using `array[index]` and are mutable by default (unless the array is declared with the const keyword, `array: const type[num] = [val1, val2, val3];`)