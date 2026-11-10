#ifndef CODEDRILL_VECT2_HPP
#define CODEDRILL_VECT2_HPP

#include <iostream>

class vect2 {
    int coordinates[2];
public:
    vect2();
    vect2(int x, int y);
    int& operator[](int index);
    int operator[](int index) const;
    vect2& operator+=(const vect2& rhs);
    vect2& operator-=(const vect2& rhs);
    vect2& operator*=(int factor);
    vect2 operator+(const vect2& rhs) const;
    vect2 operator-(const vect2& rhs) const;
    vect2 operator-() const;
    vect2 operator*(int factor) const;
    vect2& operator++();
    vect2 operator++(int);
    vect2& operator--();
    vect2 operator--(int);
    bool operator==(const vect2& rhs) const;
    bool operator!=(const vect2& rhs) const;
};

vect2 operator*(int factor, const vect2& value);
std::ostream& operator<<(std::ostream& out, const vect2& value);
#endif
