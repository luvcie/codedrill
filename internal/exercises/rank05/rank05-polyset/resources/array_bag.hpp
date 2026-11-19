#pragma once
#include "bag.hpp"

class array_bag : virtual public bag {
protected:
    int* data;
    int size;
public:
    array_bag();
    array_bag(const array_bag& source);
    array_bag& operator=(const array_bag& source);
    ~array_bag();
    void insert(int value);
    void insert(int* values, int count);
    void print() const;
    void clear();
};
