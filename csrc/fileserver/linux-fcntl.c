#include "server.h"
#include <linux/fcntl.h>

// <linux/fcntl.h> conflicts with <fcntl.h> ,
// so write the number in server.h and check it here

static_assert(FileServer_AT_EMPTY_PATH_ == AT_EMPTY_PATH, "");
