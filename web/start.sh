#!/bin/bash

die() {
    echo "$0 failed, keeping container alive for debugging..."
    while true; do sleep 10; done
}

if [ "$(id -u)" != "1000" ]; then
    echo "ERROR: Script must run as the steam user (uid 1000)"
    die
fi

# --- Build launch arguments ---

ACTUAL_PORT="${PORT:-27015}"
ARGS="-port ${ACTUAL_PORT}"

[ -n "${SERVER_PASSWORD}" ] && ARGS="${ARGS} +sv_password ${SERVER_PASSWORD}"
[ -n "${MAP}" ]             && ARGS="${ARGS} +map ${MAP}"
ARGS="${ARGS} +rcon_password ddsm_rcon_secret"

ARGS="-dedicated -usercon -ip 0.0.0.0 -convars_visible_by_default -allow_no_lobby_connect -novid ${ARGS}"

# --- Validate game directory ---

DEADLOCK_DIR=/app/Deadlock
DEADLOCK_EXE="${DEADLOCK_DIR}/game/bin/win64/deadlock.exe"
DEADWORKS_EXE="${DEADLOCK_DIR}/game/bin/win64/deadworks.exe"

mkdir -p "${DEADLOCK_DIR}"
DIR_PERM=$(stat -c "%u:%g:%a" "${DEADLOCK_DIR}")
if [ "${DIR_PERM}" != "1000:1000:755" ]; then
    echo "ERROR: ${DEADLOCK_DIR} has unexpected permissions ${DIR_PERM} (expected 1000:1000:755)"
    die
fi

# --- Download or update game files ---

if [ -f "${DEADLOCK_EXE}" ] && [ "${SKIP_UPDATE}" = "1" ]; then
    echo "Game installed and SKIP_UPDATE=1, skipping SteamCMD"
elif [ -n "${STEAM_LOGIN}" ]; then
    echo "Updating game files via SteamCMD..."
    STEAMCMD="${STEAM_HOME}/steamcmd/steamcmd.sh"
    ${STEAMCMD} \
        +@sSteamCmdForcePlatformType windows \
        +force_install_dir "${DEADLOCK_DIR}" \
        +login "${STEAM_LOGIN}" "${STEAM_PASSWORD}" "${STEAM_2FA_CODE}" \
        +app_update "${APPID}" validate \
        +quit || die
else
    echo "No STEAM_LOGIN set and game not installed"
    die
fi

if [ ! -f "${DEADLOCK_EXE}" ]; then
    echo "ERROR: ${DEADLOCK_EXE} not found after install"
    die
fi

# --- Overlay Deadworks files into the game directory ---
# Deadworks is bundled in the image at /opt/deadworks. Re-applied on every
# start so SteamCMD validation can't strip it.

DEADWORKS_SRC="${DEADWORKS_DIR:-/opt/deadworks}"
if [ -d "${DEADWORKS_SRC}/game" ]; then
    echo "Applying Deadworks framework from ${DEADWORKS_SRC}..."
    cp -rf "${DEADWORKS_SRC}/game/." "${DEADLOCK_DIR}/game/"
else
    echo "ERROR: Deadworks files not found at ${DEADWORKS_SRC}"
    die
fi

if [ ! -f "${DEADWORKS_EXE}" ]; then
    echo "ERROR: ${DEADWORKS_EXE} not found after Deadworks overlay"
    die
fi

# --- Launch server via deadworks.exe ---

CMD="${PROTON} run ${DEADWORKS_EXE} ${ARGS}"
echo "Starting Deadworks server: ${CMD}"
exec ${CMD}
