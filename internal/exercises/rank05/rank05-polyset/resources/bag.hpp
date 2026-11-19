#pragma once

class bag {
public:
    virtual void insert(int value) = 0;
    virtual void insert(int* values, int count) = 0;
    virtual void print() const = 0;
    virtual void clear() = 0;
};
