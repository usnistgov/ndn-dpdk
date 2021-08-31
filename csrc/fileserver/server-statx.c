#include "server.h"
#include <linux/fcntl.h>

// <linux/fcntl.h> conflicts with <fcntl.h> so this has to be a separate translation unit

const unsigned FileServer_StatxFlags_ = AT_EMPTY_PATH;
