import TaskManagerClient from 'src/api/TaskManagerClient';

export default ({ app }) => {
  app.config.globalProperties.$taskManagerClient = new TaskManagerClient('http://localhost:8050/api');
};
