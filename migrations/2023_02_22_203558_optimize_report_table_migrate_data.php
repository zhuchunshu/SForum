<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use Hyperf\Database\Migrations\Migration;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Schema\Schema;

class OptimizeReportTableMigrateData extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        foreach (\App\Plugins\Core\src\Models\Report::all() as $report) {
            $post = \App\Plugins\Core\src\Models\Post::find($report->post_id);
            if ($post->comment_id) {
                // 评论
                \Hyperf\DbConnection\Db::table('topic_comment')->where('id', $post->comment_id)->update([
                    'status' => 'report',
                ]);
            }else{
                if ($post->topic_id){
                    // 主题
                    \Hyperf\DbConnection\Db::table('topic')->where('id', $post->topic_id)->update([
                        'status' => 'report',
                    ]);
                }
            }
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('report', function (Blueprint $table) {
        });
    }
}
