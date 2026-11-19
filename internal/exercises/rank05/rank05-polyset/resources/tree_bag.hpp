#pragma once
#include "bag.hpp"

class tree_bag : virtual public bag {
protected:
    struct node { node* l; node* r; int value; };
    node* tree;
private:
    static node* clone(const node* root);
    static void destroy(node* root);
    static void print_branch(const node* root);
public:
    tree_bag();
    tree_bag(const tree_bag& source);
    tree_bag& operator=(const tree_bag& source);
    ~tree_bag();
    node* extract_tree();
    void set_tree(node* replacement);
    void insert(int value);
    void insert(int* values, int count);
    void print() const;
    void clear();
};
