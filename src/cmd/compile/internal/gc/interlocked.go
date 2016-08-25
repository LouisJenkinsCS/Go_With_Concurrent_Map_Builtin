package gc

func interlockednmagic(nn *Node) *Node {
	var nFn *Node
	fn := nn.Left
	args := nn.List

	if fn == nil || fn.Op != ONAME {
		return nil
	}

	s := fn.Sym
	if s == nil || s.Name != "Interlocked" {
		return nil
	}

	if s.Pkg.Name != "sync" {
		return nil
	}

	if args.Len() != 4 {
		Yyerror("Received %v arguments, expected 4", args.Len())
	}

	map_ := args.First()
	t := map_.Type

	// If the value is an indirect type, we have to make to sure substitute types differently
	if t.MapType().Val.IsPtr() {
		nFn = syslook("interlockedPtr")
	} else {
		nFn = syslook("interlocked")
	}

	key := args.Second()
	create := args.Slice()[2]
	callback := args.Slice()[3]
	key = typecheck(key, Erv)
	create = typecheck(create, Erv)
	callback = typecheck(callback, Erv)
	key = walkexpr(key, &nn.Ninit)
	create = walkexpr(create, &nn.Ninit)
	callback = walkexpr(callback, &nn.Ninit)

	nFn = substArgTypes(nFn, t.Key(), t.Val(), t.Key(), t.Val())
	nFn = mkcall1(nFn, Types[TBOOL], nil, typename(t), args.First(), Nod(OADDR, args.Second(), nil), args.Slice()[2], args.Slice()[3])
	nFn = walkstmt(nFn)

	return nFn
}

func isinterlocked(nn *Node) bool {
	fn := nn.Left
	args := nn.List

	if fn == nil || fn.Op != ONAME {
		return false
	}

	s := fn.Sym
	if s == nil || s.Name != "Interlocked" {
		return false
	}

	if s.Pkg.Name != "sync" {
		return false
	}

	if args.Len() != 4 {
		Yyerror("Received %v arguments, expected 4", args.Len())
	}

	// Make sure its arguments are marked as used.
	for _, node := range args.Slice() {
		node.Used = true
	}

	return true
}
