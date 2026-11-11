# vect2

## Original subject

The supplied local `vect2.subject.txt` is reproduced verbatim below, including
its provided main and notes.

```text
================================================================================
Assignment name  : vect2
Expected files   : vect2.hpp, vect2.cpp
Allowed functions: n/a
================================================================================

Create a class vect2 representing a mathematical vector of dimension 2
containing ints.

It must support addition, subtraction and multiplication by a scalar (such that
the expression (vect2(2, 2) * 2 == vect2(4, 4)) is true).

The operator [] can be used to access the components of your vector (with 0 and
1). No bound checking is required.

Your class must be printable with the << operator, such that:

        std::cout << v << std::endl;

produces the same output as:

        std::cout << "{" << v[0] << ", " << v[1] << "}" << std::endl;

You will find a main in this directory that must work with your class.

--------------------------------------------------------------------------------
[ provided main ]
--------------------------------------------------------------------------------

#include "vect2.hpp"
#include <iostream>

int main()
{
    vect2 v1;            // 0, 0
    vect2 v2(1, 2);      // 1, 2
    const vect2 v3(v2);  // 1, 2
    vect2 v4 = v2;       // 1, 2

    std::cout << "v1: " << v1 << std::endl;
    std::cout << "v1: " << "{" << v1[0] << ", " << v1[1] << "}" << std::endl;
    std::cout << "v2: " << v2 << std::endl;
    std::cout << "v3: " << v3 << std::endl;
    std::cout << "v4: " << v4 << std::endl;
    std::cout << v4++ << std::endl;   // 2, 3
    std::cout << ++v4 << std::endl;   // 3, 4
    std::cout << v4-- << std::endl;   // 2, 3
    std::cout << --v4 << std::endl;   // 1, 2

    v2 += v3;            // 2, 4
    v1 -= v2;            // -2, -4
    v2 = v3 + v3 * 2;    // 3, 6
    v2 = 3 * v2;         // 9, 18
    v2 += v2 += v3;      // 20, 40
    v1 *= 42;            // -84, -168
    v1 = v1 - v1 + v1;

    std::cout << "v1: " << v1 << std::endl;
    std::cout << "v2: " << v2 << std::endl;
    std::cout << "-v2: " << -v2 << std::endl;
    std::cout << "v1[1]: " << v1[1] << std::endl;
    v1[1] = 12;
    std::cout << "v1[1]: " << v1[1] << std::endl;
    std::cout << "v3[1]: " << v3[1] << std::endl;
    std::cout << "v1 == v3: " << (v1 == v3) << std::endl;
    std::cout << "v1 == v1: " << (v1 == v1) << std::endl;
    std::cout << "v1 != v3: " << (v1 != v3) << std::endl;
    std::cout << "v1 != v1: " << (v1 != v1) << std::endl;

    return (0);
}

--------------------------------------------------------------------------------
[ notes ]
--------------------------------------------------------------------------------
- Operators exercised: default ctor, ctor(int,int), copy ctor, [] (const for
  reading and non-const returning a reference so v[1] = 12 works), + - and the
  scalar *, += -= *=, unary -, pre/post ++ and --, == and !=, and <<.
- Multiplication is by a SCALAR only (both v * k and k * v). There is no
  vector * vector. The non-member operator*(int, const vect2&) handles 3 * v.
```

## Codedrill clarifications

- `main.cpp` matches the provided main above. `test_vect2.cpp` contains separate
  Codedrill regression checks; the test command compiles and runs both programs.
- The original postfix comments describe the updated object, not the printed
  return value. The four increment/decrement lines print `{1, 2}`, `{3, 4}`,
  `{3, 4}`, and `{1, 2}`. Postfix returns the previous value; prefix returns the
  updated object by reference. The original comments are preserved unchanged.
- Compound assignments return the object by reference, so `v2 += v2 += v3`
  produces `{20, 40}`. Const indexing permits reading; mutable indexing returns
  a reference so `v1[1] = 12` works.
- Compiler-generated copy construction, copy assignment, and destruction are
  sufficient for a class that stores two ints. No handwritten canonical-form
  members or vector-by-vector multiplication are required by this subject.

Passing these practice tests does not certify compliance with a particular
school exam instance or private grader.
