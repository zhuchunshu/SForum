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

class OptimizeReportTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        // 创建一个字段
        Schema::table('report', function (Blueprint $table) {
            $table->bigInteger('post_id')->nullable();
        });
        // 迁移评论
        foreach (\App\Plugins\Core\src\Models\Report::query()->where('type', 'comment')->get() as $report) {
            $id = $report->_id;
            if (\App\Plugins\Comment\src\Model\TopicComment::where('id', $id)->exists()) {
                // 评论存在
                $post_id = \App\Plugins\Comment\src\Model\TopicComment::where('id', $id)->value('post_id');
                go(function () use ($report, $post_id) {
                    \App\Plugins\Core\src\Models\Report::query()->where('id', $report->id)->update(['post_id' => $post_id]);
                });
            } else {
                // 评论消失了
                go(function () use ($report) {
                    \App\Plugins\Core\src\Models\Report::destroy($report->id);
                });
            }
        }

        // 迁移主题
        foreach (\App\Plugins\Core\src\Models\Report::query()->where('type', 'topic')->get() as $report) {
            $id = $report->_id;
            if (\App\Plugins\Topic\src\Models\Topic::where('id', $id)->exists()) {
                // 主题存在
                $post_id = \App\Plugins\Topic\src\Models\Topic::where('id', $id)->value('post_id');
                go(function () use ($report, $post_id) {
                    \App\Plugins\Core\src\Models\Report::query()->where('id', $report->id)->update(['post_id' => $post_id]);
                });
            } else {
                // 主题消失了
                go(function () use ($report) {
                    \App\Plugins\Core\src\Models\Report::destroy($report->id);
                });
            }
        }

        // 迁移完毕，删除字段
        Schema::table('report', function (Blueprint $table) {
            $table->dropColumn(['_id', 'type']);
        });
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
