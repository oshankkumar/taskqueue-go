<template>
    <div class="q-pa-md">
        <q-layout view="hHh Lpr lFf">
            <q-header elevated>
                <q-toolbar>
                    <q-btn flat @click="drawer = !drawer" round dense icon="menu" aria-label="Menu"/>
                    <q-avatar><img src="icons/favicon.svg" alt="Logo"></q-avatar>
                    <q-toolbar-title @click="this.$router.push('/')">TaskQueue Dashboard</q-toolbar-title>
                    <q-space/>
                    <div class="q-gutter-sm row items-center no-wrap">
                        <q-btn flat round dense icon="fab fa-github" type="a"
                               href="https://github.com/oshankkumar/taskqueue-go" target="_blank">
                        </q-btn>
                    </div>
                </q-toolbar>
            </q-header>

            <q-drawer
                v-model="drawer"
                show-if-above
                bordered
                content-class="bg-grey-2"
                :mini="miniState"
                @mouseenter="miniState = false"
                @mouseleave="miniState = true"
            >
                <q-list>
                    <template v-for="(menuItem, index) in menuList" :key="index">
                        <q-expansion-item
                            v-if="menuItem.submenu"
                            :icon="menuItem.icon"
                            :label="menuItem.label"
                        >
                            <q-list>
                                <template v-for="(submenu, index) in menuItem.submenu" :key="index">
                                    <q-item clickable :active="isActive(submenu.route)" v-ripple
                                            @click="navigateTo(submenu)">
                                        <q-item-section avatar>
                                            <q-icon :name="submenu.icon" :color="submenu.iconColor"/>
                                        </q-item-section>
                                        <q-item-section>
                                            {{ submenu.label }}
                                        </q-item-section>
                                    </q-item>
                                    <q-separator :key="'sep' + index" v-if="submenu.separator"/>
                                </template>
                            </q-list>
                        </q-expansion-item>
                        <q-item v-else clickable :active="isActive(menuItem.route)" v-ripple
                                @click="navigateTo(menuItem)">
                            <q-item-section avatar>
                                <q-icon :name="menuItem.icon" :color="menuItem.iconColor"/>
                            </q-item-section>
                            <q-item-section>
                                {{ menuItem.label }}
                            </q-item-section>
                        </q-item>
                        <q-separator :key="'sep' + index" v-if="menuItem.separator"/>
                    </template>
                </q-list>
            </q-drawer>

            <q-page-container>
                <router-view/>
            </q-page-container>
        </q-layout>
    </div>
</template>

<script>

const menuList = [
    {
        icon: 'home',
        label: 'Home',
        separator: true,
        route: '/home',
        active: false,
    },
    {
        icon: 'list',
        label: 'Queues',
        separator: false,
        submenu: [
            {
                icon: 'pending',
                label: 'Pending',
                iconColor: 'secondary',
                separator: false,
                route: '/pending-queues',
                active: false,
            },
            {
                icon: 'cancel',
                label: 'Dead',
                iconColor: 'negative',
                separator: false,
                route: '/dead-queues',
                active: false,
            },
        ]
    },
    {
        icon: 'send',
        label: 'Submit Job',
        separator: false,
        route: '/submit-job',
        active: false,
    },
]

export default {
    data() {
        return {
            drawer: false,
            menuList: menuList,
            miniState: true
        }
    },
    methods: {
        isActive(route) {
            return this.$route.path === route; // Check if the current route matches
        },
        navigateTo(menuItem) {
            if (menuItem.route) {
                this.$router.push(menuItem.route);
            }
        },
    },
}
</script>


