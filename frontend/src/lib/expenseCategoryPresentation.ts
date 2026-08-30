import {
    mdiAirplane,
    mdiBasketOutline,
    mdiBedOutline,
    mdiBikeFast,
    mdiBottleWineOutline,
    mdiBowlMixOutline,
    mdiBroom,
    mdiBriefcaseOutline,
    mdiBus,
    mdiCalculatorVariantOutline,
    mdiCar,
    mdiCarWrench,
    mdiCash,
    mdiDogSide,
    mdiFire,
    mdiGamepadVariant,
    mdiGasStation,
    mdiGiftOutline,
    mdiHammerWrench,
    mdiHomeCityOutline,
    mdiHumanMaleChild,
    mdiLaptop,
    mdiLightningBolt,
    mdiMovieOpen,
    mdiMusicNote,
    mdiParking,
    mdiReceiptText,
    mdiRoomServiceOutline,
    mdiSchoolOutline,
    mdiShieldCheckOutline,
    mdiSilverwareForkKnife,
    mdiSoccer,
    mdiSofaSingleOutline,
    mdiSprayBottle,
    mdiStethoscope,
    mdiTagOutline,
    mdiTaxi,
    mdiTicketOutline,
    mdiToilet,
    mdiTrashCanOutline,
    mdiTshirtCrewOutline,
    mdiWaterOutline,
    mdiWifi,
} from "@mdi/js";

export interface ExpenseCategoryPresentation {
    icon: string;
    iconClassName: string;
    label: string;
}

type ExpenseTypePresentation = Pick<ExpenseCategoryPresentation, "icon">;

const categoryPresentations: Record<string, ExpenseCategoryPresentation> = {
    entertainment: {
        icon: mdiGamepadVariant,
        iconClassName: "bg-violet-100 text-violet-700",
        label: "Entertainment",
    },
    "food and drink": {
        icon: mdiSilverwareForkKnife,
        iconClassName: "bg-emerald-100 text-emerald-700",
        label: "Food and drink",
    },
    home: {
        icon: mdiHomeCityOutline,
        iconClassName: "bg-amber-100 text-amber-700",
        label: "Home",
    },
    life: {
        icon: mdiShieldCheckOutline,
        iconClassName: "bg-orange-100 text-orange-700",
        label: "Life",
    },
    transportation: {
        icon: mdiCar,
        iconClassName: "bg-rose-100 text-rose-700",
        label: "Transportation",
    },
    uncategorized: {
        icon: mdiReceiptText,
        iconClassName: "bg-slate-100 text-slate-700",
        label: "Uncategorized",
    },
    utilities: {
        icon: mdiReceiptText,
        iconClassName: "bg-sky-100 text-sky-700",
        label: "Utilities",
    },
};

const typePresentations: Record<string, ExpenseTypePresentation> = {
    "entertainment:games": { icon: mdiGamepadVariant },
    "entertainment:movies": { icon: mdiMovieOpen },
    "entertainment:music": { icon: mdiMusicNote },
    "entertainment:other": { icon: mdiTicketOutline },
    "entertainment:sport": { icon: mdiSoccer },
    "food and drink:dining out": { icon: mdiSilverwareForkKnife },
    "food and drink:groceries": { icon: mdiBasketOutline },
    "food and drink:liquor": { icon: mdiBottleWineOutline },
    "food and drink:other": { icon: mdiBowlMixOutline },
    "home:electronics": { icon: mdiLaptop },
    "home:furniture": { icon: mdiSofaSingleOutline },
    "home:household supplies": { icon: mdiSprayBottle },
    "home:maintenance": { icon: mdiHammerWrench },
    "home:mortgage": { icon: mdiHomeCityOutline },
    "home:other": { icon: mdiToilet },
    "home:pets": { icon: mdiDogSide },
    "home:rent": { icon: mdiCash },
    "home:services": { icon: mdiRoomServiceOutline },
    "life:childcare": { icon: mdiHumanMaleChild },
    "life:clothing": { icon: mdiTshirtCrewOutline },
    "life:education": { icon: mdiSchoolOutline },
    "life:gifts": { icon: mdiGiftOutline },
    "life:insurance": { icon: mdiShieldCheckOutline },
    "life:medical expenses": { icon: mdiStethoscope },
    "life:other": { icon: mdiBriefcaseOutline },
    "life:taxes": { icon: mdiCalculatorVariantOutline },
    "transportation:bicycle": { icon: mdiBikeFast },
    "transportation:bus/train": { icon: mdiBus },
    "transportation:car": { icon: mdiCar },
    "transportation:gas/fuel": { icon: mdiGasStation },
    "transportation:hotel": { icon: mdiBedOutline },
    "transportation:other": { icon: mdiCarWrench },
    "transportation:parking": { icon: mdiParking },
    "transportation:plane": { icon: mdiAirplane },
    "transportation:taxi": { icon: mdiTaxi },
    "uncategorized:general": { icon: mdiReceiptText },
    "utilities:cleaning": { icon: mdiBroom },
    "utilities:electricity": { icon: mdiLightningBolt },
    "utilities:heat/gas": { icon: mdiFire },
    "utilities:other": { icon: mdiTagOutline },
    "utilities:trash": { icon: mdiTrashCanOutline },
    "utilities:tv/phone/internet": { icon: mdiWifi },
    "utilities:water": { icon: mdiWaterOutline },
};

const fallbackPresentation: ExpenseCategoryPresentation = {
    icon: mdiTagOutline,
    iconClassName: "bg-slate-100 text-slate-700",
    label: "Other",
};

function normalized(value?: string) {
    return value?.trim().toLowerCase() ?? "";
}

export function getExpenseCategoryPresentation(category?: string) {
    return categoryPresentations[normalized(category)] ?? fallbackPresentation;
}

export function getExpenseTypePresentation(category?: string, type?: string) {
    const categoryPresentation = getExpenseCategoryPresentation(category);
    const typePresentation =
        typePresentations[`${normalized(category)}:${normalized(type)}`];

    return {
        ...categoryPresentation,
        icon: typePresentation?.icon ?? categoryPresentation.icon,
    };
}
