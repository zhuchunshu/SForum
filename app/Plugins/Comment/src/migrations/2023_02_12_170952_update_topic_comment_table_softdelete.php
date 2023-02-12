<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateTopicCommentTableSoftdelete extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('topic_comment', function (Blueprint $table) {
            $table->softDeletes();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('topic_comment', function (Blueprint $table) {
            //
        });
    }
}
