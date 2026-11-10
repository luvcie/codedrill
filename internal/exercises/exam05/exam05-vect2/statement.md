# vect2

Build a value type named `vect2` for two integer coordinates. Submit `vect2.hpp`
and `vect2.cpp`. The supplied `main.cpp` is Codedrill's test, not a school file.

Default construction produces `(0, 0)`; a two-int constructor supplies the
coordinates. Copies and assignments must remain independent of their source.
Support reading either coordinate on const objects and modifying it on mutable
objects through `operator[]`. Only indices 0 and 1 are used; bounds checks are
unnecessary.

Implement component-wise `+`, binary `-`, unary `-`, `+=`, `-=`, multiplication
by an int in either order, and `*=`. Prefix `++`/`--` change both coordinates
and return the object by reference; postfix versions return the previous value.
Compound assignments also return the object by reference. Equality compares
both coordinates. Provide `==` and `!=`.

Stream insertion writes `{x, y}` to the supplied stream and returns that stream.
Storage layout, helper methods, and handwritten versus generated copy/destructor
members are your choice. Vector-by-vector multiplication is not required.

This is an independently written Exam 05 practice exercise. Passing its tests
does not certify compliance with a particular school exam instance.
