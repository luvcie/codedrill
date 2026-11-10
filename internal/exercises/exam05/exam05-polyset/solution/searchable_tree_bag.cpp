#include "searchable_tree_bag.hpp"

searchable_tree_bag::searchable_tree_bag() = default;
searchable_tree_bag::searchable_tree_bag(const searchable_tree_bag& source) : tree_bag(source) {}
searchable_tree_bag& searchable_tree_bag::operator=(const searchable_tree_bag& source) {
    tree_bag::operator=(source);
    return *this;
}
searchable_tree_bag::~searchable_tree_bag() = default;
bool searchable_tree_bag::has(int value) const {
    const node* position = tree;
    while (position && position->value != value) {
        position = value < position->value ? position->l : position->r;
    }
    return position != nullptr;
}
