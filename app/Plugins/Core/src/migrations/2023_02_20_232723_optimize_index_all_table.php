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
use Hyperf\DB\DB;
use Hyperf\Context\ApplicationContext;

class OptimizeIndexAllTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        $db = ApplicationContext::getContainer()->get(DB::class);

        // 处理全部user_id字段
        $tables = Schema::getAllTables();
        foreach ($tables as $table) {
            $name = $table[0];
            if (Schema::hasColumn($name, 'user_id') && Schema::getColumnType($name, 'user_id') !== 'bigint') {
                $db->query('ALTER TABLE ' . $name . ' MODIFY COLUMN user_id BIGINT;');
                Schema::table($name, function (Blueprint $table) {
                    $table->index('user_id');
                });
            }
        }

        // topic表
        if (Schema::hasColumn('topic', 'tag_id') && Schema::getColumnType('topic', 'tag_id') !== 'bigint') {
            $db->query('ALTER TABLE topic MODIFY COLUMN user_id BIGINT;');
            Schema::table('topic', function (Blueprint $table) {
                $table->index('tag_id');
            });
        }

        // topic-comment 表
        if (Schema::hasColumn('topic_comment', 'topic_id') && Schema::getColumnType('topic_comment', 'topic_id') !== 'bigint') {
            $db->query('ALTER TABLE topic_comment MODIFY COLUMN topic_id BIGINT;');
            Schema::table('topic_comment', function (Blueprint $table) {
                $table->index('topic_id');
            });
        }

        // users 表
        if (Schema::hasColumn('users', 'group_id') && Schema::getColumnType('users', 'class_id') !== 'bigint') {
            $db->query('ALTER TABLE users MODIFY COLUMN class_id BIGINT;');
            Schema::table('users', function (Blueprint $table) {
                $table->index('class_id');
            });
        }
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
    }
}
