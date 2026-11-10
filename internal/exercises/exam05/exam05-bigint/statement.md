# bigint

Implement an unsigned integer value type whose precision grows with available
memory. Submit `bigint.hpp` and `bigint.cpp`; `main.cpp` is a Codedrill test.

Support default construction as zero, implicit construction from `unsigned int`,
independent copying/assignment, addition (`+`, `+=`), prefix/postfix increment,
and all six relational operators. Do not limit stored values to a machine integer.

The shift operators move decimal digits, not bits: shifting 73 left by 2 gives
7300, and shifting 7300 right by 3 gives 7. Shift counts are `bigint` values;
integer counts work through conversion. Implement `<<`, `>>`, `<<=`, and `>>=`.
Right shifting away every digit gives zero. Shifting zero keeps it zero.
Compound operators and prefix increment return the modified object by reference;
postfix increment returns its former value. Stream insertion prints decimal
digits without leading zeroes and returns the supplied stream.

Subtraction, division, and multiplication are outside this exercise. Tests use
feasible left shifts; a right shift larger than any machine integer still gives
zero. The supplied reference reports impossible left allocations with standard
C++ exceptions, without defining an additional school requirement.

This practice specification and its tests are independently written; consult
the actual exam screen for the school assignment's requirements.
