#ifndef VECT2_HPP
#define VECT2_HPP

#include <iostream>

class vect2 {
  int coordinates[2];

public:

  // 1. Constructors
  vect2() {
    coordinates[0] = 0;
    coordinates[1] = 0;
  }

  vect2(int x, int y) {
    coordinates[0] = x;
    coordinates[1] = y;
  }

  // 2. Indexing
  int &operator[](int index) { return coordinates[index]; }

  int operator[](int index) const { return coordinates[index]; }

  // 3. Modify the current vector
  vect2 &operator+=(const vect2 &rhs) {
    coordinates[0] += rhs[0];
    coordinates[1] += rhs[1];
    return *this;
  }

  vect2 &operator-=(const vect2 &rhs) {
    coordinates[0] -= rhs[0];
    coordinates[1] -= rhs[1];
    return *this;
  }

  vect2 &operator*=(int factor) {
    coordinates[0] *= factor;
    coordinates[1] *= factor;
    return *this;
  }

  // 4. Return a new vector
  vect2 operator+(const vect2 &rhs) const {
    return vect2(coordinates[0] + rhs[0], coordinates[1] + rhs[1]);
  }

  vect2 operator-(const vect2 &rhs) const {
    return vect2(coordinates[0] - rhs[0], coordinates[1] - rhs[1]);
  }

  vect2 operator-() const {
    return vect2(-coordinates[0], -coordinates[1]);
  }

  vect2 operator*(int factor) const {
    return vect2(coordinates[0] * factor, coordinates[1] * factor);
  }

  // 5. Increment and decrement
  vect2 &operator++() { return *this += vect2(1, 1); }

  vect2 operator++(int) {
    vect2 old = *this;
    ++*this;
    return old;
  }

  vect2 &operator--() { return *this -= vect2(1, 1); }

  vect2 operator--(int) {
    vect2 old = *this;
    --*this;
    return old;
  }

  // 6. Comparisons
  bool operator==(const vect2 &rhs) const {
    return coordinates[0] == rhs[0] && coordinates[1] == rhs[1];
  }

  bool operator!=(const vect2 &rhs) const {
    return !(*this == rhs);
  }
};

// 7. Operators outside the class
inline vect2 operator*(int factor, const vect2 &value) {
  return value * factor;
}

inline std::ostream &operator<<(std::ostream &out, const vect2 &value) {
  return out << '{' << value[0] << ", " << value[1] << '}';
}

#endif
