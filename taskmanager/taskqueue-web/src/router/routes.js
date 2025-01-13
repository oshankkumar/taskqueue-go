const routes = [
    {
        path: '/',
        component: () => import('layouts/MainLayout.vue'),
        children: [
            {path: '/', component: () => import('pages/Home.vue')},
            {path: '/home', component: () => import('pages/Home.vue')},
            {path: '/pending-queues', component: () => import('pages/PendingQueues.vue')},
            {path: '/dead-queues', component: () => import('pages/DeadQueues.vue')},
            {path: '/completed-queues', component: () => import('pages/CompletedQueues.vue')},
            {path: '/pending-queues/:queue_name', component: () => import('pages/QueueDetails.vue')},
            {path: '/completed-queues/:queue_name', component: () => import('pages/QueueDetails.vue')},
            {path: '/dead-queues/:queue_name', component: () => import('pages/QueueDetails.vue')},
            {path: '/submit-job', component: () => import('pages/SubmitJob.vue')}
        ]
    },

    // Always leave this as last one,
    // but you can also remove it
    {
        path: '/:catchAll(.*)*',
        component: () => import('pages/ErrorNotFound.vue')
    }
]

export default routes
