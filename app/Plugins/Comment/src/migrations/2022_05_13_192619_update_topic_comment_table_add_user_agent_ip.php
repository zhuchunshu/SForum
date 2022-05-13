<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateTopicCommentTableAddUserAgentIp extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('topic_comment', function (Blueprint $table) {
            $table->string('user_agent')->nullable();
			$table->string('user_ip')->nullable();
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
