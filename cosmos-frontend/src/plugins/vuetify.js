import Vue from 'vue';
import Vuetify from 'vuetify/lib/framework';
import {
    VApp,
    VAutocomplete,
    VAvatar,
    VBtn,
    VCard,
    VCheckbox,
    VCol,
    VContainer,
    VDatePicker,
    VDialog,
    VDivider,
    VIcon,
    VList,
    VMain,
    VMenu,
    VNavigationDrawer,
    VPagination,
    VProgressLinear,
    VRow,
    VSelect,
    VSnackbar,
    VSpacer,
    VSwitch,
    VTab,
    VTabs,
    VTextField,
    VToolbar,
    VTooltip,
} from 'vuetify/lib';
import {
    Ripple
} from 'vuetify/lib/directives';

Vue.use(Vuetify, {
    // Any new vuetify component used needs to be added here.
    // Otherwise, you'll get "min-css-extract-plugin" conflicting order errors.
    // See https://stackoverflow.com/a/64419994
    components: {
        VApp,
        VAutocomplete,
        VAvatar,
        VBtn,
        VCard,
        VCheckbox,
        VCol,
        VContainer,
        VDatePicker,
        VDialog,
        VDivider,
        VIcon,
        VList,
        VMain,
        VMenu,
        VNavigationDrawer,
        VPagination,
        VProgressLinear,
        VRow,
        VSelect,
        VSnackbar,
        VSpacer,
        VSwitch,
        VTab,
        VTabs,
        VTextField,
        VToolbar,
        VTooltip,
    },
    directives: {
        Ripple,
    },
})

export default new Vuetify({
});
