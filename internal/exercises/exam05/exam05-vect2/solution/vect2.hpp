#ifndef VECT2_HPP
#define VECT2_HPP

#include <iostream>

class vect2 {
    int coordinates[2];

public:
    // 1. Constructors
    vect2();
    vect2(int x, int y);

    // 2. Indexing
    int& operator[](int index);
    int operator[](int index) const;

    // 3. Modify the current vector
    vect2& operator+=(const vect2& rhs);
    vect2& operator-=(const vect2& rhs);
    vect2& operator*=(int factor);

    // 4. Return a new vector
    vect2 operator+(const vect2& rhs) const;
    vect2 operator-(const vect2& rhs) const;
    vect2 operator-() const;
    vect2 operator*(int factor) const;

    // 5. Increment and decrement
    vect2& operator++();
    vect2 operator++(int);
    vect2& operator--();
    vect2 operator--(int);

    // 6. Comparisons
    bool operator==(const vect2& rhs) const;
    bool operator!=(const vect2& rhs) const;
};

// 7. Free functions
vect2 operator*(int factor, const vect2& value);
std::ostream& operator<<(std::ostream& out, const vect2& value);

#endif
