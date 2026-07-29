const state = $state({
    token: localStorage.getItem("token"),
});

export function getAuth() {
    return state;
}

export function setToken(token: string | null) {
    if (token !== null) {
        localStorage.setItem("token", token);
    } else {
        localStorage.removeItem("token");
    }

    state.token = token;
}
