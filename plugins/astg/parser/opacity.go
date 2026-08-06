// Copyright (c) 2026 Khramtsov Aleksei (seniorGolang@gmail.com).
// conditions defined in file 'LICENSE', which is part of this project source code.
package parser

import (
	"go/types"
)

func typeHasJSONCodec(typ types.Type) (ok bool) {

	if typ == nil {
		return
	}

	if hasJSONCodecMethodSet(typ) {
		return true
	}
	if _, isPtr := typ.(*types.Pointer); !isPtr {
		return hasJSONCodecMethodSet(types.NewPointer(typ))
	}
	return
}

func hasJSONCodecMethodSet(typ types.Type) (ok bool) {

	mset := types.NewMethodSet(typ)
	for method := range mset.Methods() {
		if method == nil || method.Obj() == nil {
			continue
		}
		name := method.Obj().Name()
		sig, isSig := method.Type().(*types.Signature)
		if !isSig {
			continue
		}
		switch name {
		case "MarshalJSON":
			if isMarshalJSONSignature(sig) {
				return true
			}
		case "UnmarshalJSON":
			if isUnmarshalJSONSignature(sig) {
				return true
			}
		}
	}
	return
}

func isMarshalJSONSignature(sig *types.Signature) (ok bool) {

	if sig.Params().Len() != 0 || sig.Results().Len() != 2 {
		return
	}
	if !isByteSliceType(sig.Results().At(0).Type()) {
		return
	}
	return isErrorType(sig.Results().At(1).Type())
}

func isUnmarshalJSONSignature(sig *types.Signature) (ok bool) {

	if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
		return
	}
	if !isByteSliceType(sig.Params().At(0).Type()) {
		return
	}
	return isErrorType(sig.Results().At(0).Type())
}

func isByteSliceType(typ types.Type) (ok bool) {

	slice, isSlice := typ.(*types.Slice)
	if !isSlice {
		return
	}
	basic, isBasic := slice.Elem().(*types.Basic)
	if !isBasic {
		return
	}
	kind := basic.Kind()
	return kind == types.Byte || kind == types.Uint8
}

func isErrorType(typ types.Type) (ok bool) {

	return types.Identical(typ, types.Universe.Lookup("error").Type())
}
