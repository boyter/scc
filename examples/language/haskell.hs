module RBTree
    ( RBTree
    , empty
    , insert
    , member
    , fromList
    ) where

data Color = Red | Black
    deriving (Show, Eq)

data RBTree a = Empty 
              | Node Color (RBTree a) a (RBTree a)
    deriving (Show, Eq)

empty :: RBTree a
empty = Empty

member :: (Ord a) => a -> RBTree a -> Bool
member _ Empty = False
member x (Node _ left val right) =
    case compare x val of
        LT -> member x left
        GT -> member x right
        EQ -> True

insert :: (Ord a) => a -> RBTree a -> RBTree a
insert x s = makeBlack (ins s)
  where
    makeBlack Empty = Empty -- never come here
    makeBlack (Node _ l v r) = Node Black l v r

    ins Empty = Node Red Empty x Empty
    ins (Node color l v r)
        | x < v     = balance color (ins l) v r
        | x > v     = balance color l v (ins r)
        | otherwise = Node color l v r  -- discard duplicated value

balance :: Color -> RBTree a -> a -> RBTree a -> RBTree a
balance Black (Node Red (Node Red a x b) y c) z d =
    Node Red (Node Black a x b) y (Node Black c z d)
balance Black (Node Red a x (Node Red b y c)) z d =
    Node Red (Node Black a x b) y (Node Black c z d)
balance Black a x (Node Red (Node Red b y c) z d) =
    Node Red (Node Black a x b) y (Node Black c z d)
balance Black a x (Node Red b y (Node Red c z d)) =
    Node Red (Node Black a x b) y (Node Black c z d)
balance color l v r = Node color l v r

fromList :: (Ord a) => [a] -> RBTree a
fromList = foldr insert empty
