<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateUsersCollectionsTableR extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('users_collections', function (Blueprint $table) {
            // 删除字段
            if (Schema::hasColumn('users_collections', 'title')) {
                $table->dropColumn('title');
            }
            if (Schema::hasColumn('users_collections', 'content')) {
                $table->dropColumn('content');
            }
            if (Schema::hasColumn('users_collections', 'action')) {
                $table->dropColumn('action');
            }
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('users_collections', function (Blueprint $table) {
            //
        });
    }
}
