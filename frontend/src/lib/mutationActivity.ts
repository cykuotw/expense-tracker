type MutationActivityListener = (active: boolean) => void;

let activeMutationCount = 0;
const listeners = new Set<MutationActivityListener>();

function notifyMutationActivity() {
    const active = activeMutationCount > 0;
    listeners.forEach((listener) => listener(active));
}

export function beginMutationActivity() {
    activeMutationCount += 1;
    notifyMutationActivity();

    let finished = false;
    return () => {
        if (finished) return;
        finished = true;
        activeMutationCount = Math.max(0, activeMutationCount - 1);
        notifyMutationActivity();
    };
}

export function subscribeMutationActivity(listener: MutationActivityListener) {
    listeners.add(listener);
    listener(activeMutationCount > 0);
    return () => {
        listeners.delete(listener);
    };
}
