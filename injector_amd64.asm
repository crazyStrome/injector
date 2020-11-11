#include "textflag.h"

TEXT InjectStructPtr(SB), NOSPLIT, $0-16
    MOVQ pos+8(SP), CX
    MOVQ dat+16(SP), AX
    MOVQ AX, (CX)
    RET
    