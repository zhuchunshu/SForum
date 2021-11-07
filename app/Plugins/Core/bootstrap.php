<?php
Authority()->add("topic_edit","编辑自己的帖子");
Authority()->add("topic_create","发帖权限");
Authority()->add("topic_options","修改帖子状态(置顶,精华等)");
Authority()->add("comment_create","发布评论权限");
Authority()->add("comment_edit","修改自己的评论");

Authority()->add("admin_topic_edit","修改所有帖子");
Authority()->add("admin_comment_edit","修改所有评论");

Authority()->add("admin_view_draft_topic","预览所有(他人)草稿");
Authority()->add("admin_comment_remove","删除所有(他人)评论");
Authority()->add("comment_remove","删除自己评论");

Authority()->add("admin_comment_caina","采纳所有帖子下的评论");
Authority()->add("comment_caina","采纳自己帖子下的评论");

Authority()->add("admin_topic_delete","删除所有帖子");
Authority()->add("topic_delete","删除自己帖子");
Authority()->add("report","举报帖子");
Authority()->add("report_comment","举报评论");
Authority()->add("admin_report","受理举报");

Itf()->add("core_auth_selected","topic_edit","topic_edit");
Itf()->add("core_auth_selected","topic_create","topic_create");
Itf()->add("core_auth_selected","comment_create","comment_create");
Itf()->add("core_auth_selected","comment_edit","comment_edit");
Itf()->add("core_auth_selected","comment_remove","comment_remove");
Itf()->add("core_auth_selected","comment_caina","comment_caina");
Itf()->add("core_auth_selected","topic_delete","topic_delete");
Itf()->add("core_auth_selected","report_topic","report_topic");
Itf()->add("core_auth_selected","report_comment","report_comment");
